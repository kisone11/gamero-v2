// Package service 提供玩家游戏评测的业务逻辑层。
package service

import (
	"context"
	"time"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/storage"
)

// ===========================
// DTO
// ===========================

// CreateGameReviewReq 创建评测请求
type CreateGameReviewReq struct {
	Rating  int8   `json:"rating" binding:"required,min=1,max=5"`
	Content string `json:"content" binding:"max=1000"`
}

// ReviewItem 评测列表项
type ReviewItem struct {
	ID              uint64     `json:"id"`
	ProjectID       uint64     `json:"project_id"`
	UserID          uint64     `json:"user_id"`
	UserNickname    string     `json:"user_nickname"`
	UserAvatarURL   string     `json:"user_avatar_url"`
	Rating          int8       `json:"rating"`
	Content         string     `json:"content"`
	LikeCount       int        `json:"like_count"`
	IsLiked         bool       `json:"is_liked"` // 当前请求者是否已点赞
	ReplyContent    string     `json:"reply_content"`             // 项目作者回复
	RepliedAt       *time.Time `json:"replied_at,omitempty"`      // 回复时间
	CreatedAt       time.Time  `json:"created_at"`
}

// ===========================
// 接口
// ===========================

// ReviewService 玩家评测业务逻辑接口
type ReviewService interface {
	// CreateReview 创建评测（每个用户对每个项目只能评测一次）
	CreateReview(ctx context.Context, userID, projectID uint64, req *CreateGameReviewReq) (*ReviewItem, error)
	// UpdateReview 更新评测（本人）
	UpdateReview(ctx context.Context, userID, reviewID uint64, req *CreateGameReviewReq) (*ReviewItem, error)
	// DeleteReview 删除评测（本人）
	DeleteReview(ctx context.Context, userID, reviewID uint64) error
	// ListReviews 获取项目评测列表（分页，按点赞数/时间排序）
	ListReviews(ctx context.Context, projectID, viewerID uint64, page, pageSize int, sortBy string) ([]*ReviewItem, int64, error)
	// GetRatingSummary 获取项目评分汇总
	GetRatingSummary(ctx context.Context, projectID uint64) (*repository.ReviewRatingSummary, error)
	// LikeReview 点赞评测（幂等）
	LikeReview(ctx context.Context, userID, reviewID uint64) error
	// UnlikeReview 取消点赞
	UnlikeReview(ctx context.Context, userID, reviewID uint64) error
	// ReplyReview 项目作者回复评测（只有项目 owner 可操作）
	ReplyReview(ctx context.Context, projectID, reviewID, actorID uint64, content string) error
}

// ===========================
// 实现
// ===========================

type reviewService struct {
	reviewRepo  repository.ReviewRepository
	userRepo    repository.UserRepository
	projectRepo repository.ProjectRepository
}

// NewReviewService 创建评测服务
func NewReviewService(reviewRepo repository.ReviewRepository, userRepo repository.UserRepository, projectRepo repository.ProjectRepository) ReviewService {
	return &reviewService{reviewRepo: reviewRepo, userRepo: userRepo, projectRepo: projectRepo}
}

func presignURL(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	url, err := storage.Get().PresignedGetURL(ctx, key, 24*time.Hour)
	if err != nil {
		return storage.Get().GetPublicURL(key)
	}
	return url
}

func (s *reviewService) CreateReview(ctx context.Context, userID, projectID uint64, req *CreateGameReviewReq) (*ReviewItem, error) {
	// 验证项目是否存在
	if _, err := s.projectRepo.GetProjectByID(ctx, projectID); err != nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "项目不存在")
	}
	// 一人一评
	existing, err := s.reviewRepo.GetByUserAndProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "您已评测过该项目，请直接编辑")
	}
	review := &model.GameReview{
		ProjectID: projectID,
		UserID:    userID,
		Rating:    req.Rating,
		Content:   req.Content,
	}
	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}
	return s.toItem(ctx, review, userID)
}

func (s *reviewService) UpdateReview(ctx context.Context, userID, reviewID uint64, req *CreateGameReviewReq) (*ReviewItem, error) {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if review.UserID != userID {
		return nil, apperrors.CodeError(apperrors.CodeForbidden)
	}
	review.Rating = req.Rating
	review.Content = req.Content
	if err := s.reviewRepo.Update(ctx, review); err != nil {
		return nil, err
	}
	return s.toItem(ctx, review, userID)
}

func (s *reviewService) DeleteReview(ctx context.Context, userID, reviewID uint64) error {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if review.UserID != userID {
		return apperrors.CodeError(apperrors.CodeForbidden)
	}
	return s.reviewRepo.Delete(ctx, reviewID)
}

func (s *reviewService) ListReviews(ctx context.Context, projectID, viewerID uint64, page, pageSize int, sortBy string) ([]*ReviewItem, int64, error) {
	offset := (page - 1) * pageSize
	reviews, total, err := s.reviewRepo.List(ctx, projectID, offset, pageSize, sortBy)
	if err != nil {
		return nil, 0, err
	}
	if len(reviews) == 0 {
		return nil, total, nil
	}

	// 批量查询用户信息（避免 N+1）
	userIDs := make([]uint64, 0, len(reviews))
	for _, r := range reviews {
		userIDs = append(userIDs, r.UserID)
	}
	users, _ := s.userRepo.GetUsersByIDs(ctx, userIDs)
	userMap := make(map[uint64]*model.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 批量查询当前用户的点赞状态（避免 N+1）
	likedSet := make(map[uint64]bool)
	if viewerID > 0 {
		reviewIDs := make([]uint64, 0, len(reviews))
		for _, r := range reviews {
			reviewIDs = append(reviewIDs, r.ID)
		}
		likedSet, _ = s.reviewRepo.BatchIsLiked(ctx, reviewIDs, viewerID)
	}

	items := make([]*ReviewItem, 0, len(reviews))
	for _, r := range reviews {
		item := &ReviewItem{
			ID:           r.ID,
			ProjectID:    r.ProjectID,
			UserID:       r.UserID,
			Rating:       r.Rating,
			Content:      r.Content,
			LikeCount:    r.LikeCount,
			IsLiked:      likedSet[r.ID],
			ReplyContent: r.ReplyContent,
			RepliedAt:    r.RepliedAt,
			CreatedAt:    r.CreatedAt,
		}
		if u, ok := userMap[r.UserID]; ok {
			item.UserNickname = u.Nickname
			if u.AvatarKey != "" {
				item.UserAvatarURL = presignURL(ctx, u.AvatarKey)
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (s *reviewService) GetRatingSummary(ctx context.Context, projectID uint64) (*repository.ReviewRatingSummary, error) {
	return s.reviewRepo.GetRatingSummary(ctx, projectID)
}

func (s *reviewService) LikeReview(ctx context.Context, userID, reviewID uint64) error {
	return s.reviewRepo.UpsertLike(ctx, reviewID, userID)
}

func (s *reviewService) UnlikeReview(ctx context.Context, userID, reviewID uint64) error {
	return s.reviewRepo.DeleteLike(ctx, reviewID, userID)
}

func (s *reviewService) ReplyReview(ctx context.Context, projectID, reviewID, actorID uint64, content string) error {
	// 权限检查：只有项目 owner 可以回复
	project, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return apperrors.New(apperrors.CodeNotFound, "项目不存在")
	}
	if project.OwnerID != actorID {
		return apperrors.CodeError(apperrors.CodeForbidden)
	}
	// 确认评测存在
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}
	// 确认评测属于该项目
	if review.ProjectID != projectID {
		return apperrors.New(apperrors.CodeParamInvalid, "评测不属于该项目")
	}
	now := time.Now()
	return s.reviewRepo.ReplyReview(ctx, reviewID, content, &now)
}

// toItem 将 model.GameReview 转换为 ReviewItem（含用户信息）
func (s *reviewService) toItem(ctx context.Context, r *model.GameReview, viewerID uint64) (*ReviewItem, error) {
	item := &ReviewItem{
		ID:           r.ID,
		ProjectID:    r.ProjectID,
		UserID:       r.UserID,
		Rating:       r.Rating,
		Content:      r.Content,
		LikeCount:    r.LikeCount,
		ReplyContent: r.ReplyContent,
		RepliedAt:    r.RepliedAt,
		CreatedAt:    r.CreatedAt,
	}
	if user, err := s.userRepo.GetUserByID(ctx, r.UserID); err == nil && user != nil {
		item.UserNickname = user.Nickname
		if user.AvatarKey != "" {
			item.UserAvatarURL = presignURL(ctx, user.AvatarKey)
		}
	}
	if viewerID > 0 {
		isLiked, _ := s.reviewRepo.IsLiked(ctx, r.ID, viewerID)
		item.IsLiked = isLiked
	}
	return item, nil
}
