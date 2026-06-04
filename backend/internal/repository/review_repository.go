// Package repository 提供玩家游戏评测的数据库访问层。
package repository

import (
	"context"
	"time"

	"github.com/gamero/gamero/internal/model"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"gorm.io/gorm"
)

// ReviewRatingSummary 评分汇总（定义在 repository 包避免循环依赖）
type ReviewRatingSummary struct {
	TotalCount   int64          `json:"total_count"`
	AverageScore float64        `json:"average_score"`
	Distribution map[int8]int64 `json:"distribution"`
}

// ReviewRepository 评测数据访问接口
type ReviewRepository interface {
	// Create 创建评测
	Create(ctx context.Context, review *model.GameReview) error
	// GetByID 根据 ID 查询评测
	GetByID(ctx context.Context, id uint64) (*model.GameReview, error)
	// GetByUserAndProject 查询用户对项目的评测（不存在返回 nil,nil）
	GetByUserAndProject(ctx context.Context, userID, projectID uint64) (*model.GameReview, error)
	// Update 更新评测
	Update(ctx context.Context, review *model.GameReview) error
	// Delete 删除评测
	Delete(ctx context.Context, id uint64) error
	// List 分页查询项目评测列表
	List(ctx context.Context, projectID uint64, offset, limit int, sortBy string) ([]*model.GameReview, int64, error)
	// GetRatingSummary 获取项目评分汇总
	GetRatingSummary(ctx context.Context, projectID uint64) (*ReviewRatingSummary, error)
	// UpsertLike 点赞（幂等）
	UpsertLike(ctx context.Context, reviewID, userID uint64) error
	// DeleteLike 取消点赞
	DeleteLike(ctx context.Context, reviewID, userID uint64) error
	// IsLiked 判断是否已点赞
	IsLiked(ctx context.Context, reviewID, userID uint64) (bool, error)
	// BatchIsLiked 批量判断用户是否已点赞（reviewIDs 中哪些被 userID 点赞）
	BatchIsLiked(ctx context.Context, reviewIDs []uint64, userID uint64) (map[uint64]bool, error)
	// ReplyReview 更新评测的回复内容和回复时间
	ReplyReview(ctx context.Context, reviewID uint64, content string, repliedAt *time.Time) error
}

type reviewRepository struct {
	db *gorm.DB
}

// NewReviewRepository 创建评测仓储
func NewReviewRepository(db *gorm.DB) ReviewRepository {
	return &reviewRepository{db: db}
}

func (r *reviewRepository) Create(ctx context.Context, review *model.GameReview) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *reviewRepository) GetByID(ctx context.Context, id uint64) (*model.GameReview, error) {
	var review model.GameReview
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&review).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperrors.New(apperrors.CodeNotFound, "评测不存在")
	}
	return &review, err
}

func (r *reviewRepository) GetByUserAndProject(ctx context.Context, userID, projectID uint64) (*model.GameReview, error) {
	var review model.GameReview
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		First(&review).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &review, err
}

func (r *reviewRepository) Update(ctx context.Context, review *model.GameReview) error {
	return r.db.WithContext(ctx).Save(review).Error
}

func (r *reviewRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.GameReview{}, id).Error
}

func (r *reviewRepository) List(ctx context.Context, projectID uint64, offset, limit int, sortBy string) ([]*model.GameReview, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.GameReview{}).
		Where("project_id = ?", projectID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderClause := "created_at DESC"
	if sortBy == "helpful" {
		orderClause = "like_count DESC, created_at DESC"
	}

	var reviews []*model.GameReview
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order(orderClause).
		Offset(offset).Limit(limit).
		Find(&reviews).Error; err != nil {
		return nil, 0, err
	}
	return reviews, total, nil
}

func (r *reviewRepository) GetRatingSummary(ctx context.Context, projectID uint64) (*ReviewRatingSummary, error) {
	type row struct {
		Rating int8
		Cnt    int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Model(&model.GameReview{}).
		Select("rating, COUNT(*) as cnt").
		Where("project_id = ?", projectID).
		Group("rating").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	summary := &ReviewRatingSummary{
		Distribution: make(map[int8]int64),
	}
	var total int64
	var sum int64
	for _, row := range rows {
		summary.Distribution[row.Rating] = row.Cnt
		total += row.Cnt
		sum += int64(row.Rating) * row.Cnt
	}
	summary.TotalCount = total
	if total > 0 {
		summary.AverageScore = float64(sum) / float64(total)
	}
	return summary, nil
}

func (r *reviewRepository) UpsertLike(ctx context.Context, reviewID, userID uint64) error {
	like := &model.GameReviewLike{ReviewID: reviewID, UserID: userID}
	result := r.db.WithContext(ctx).
		Where("review_id = ? AND user_id = ?", reviewID, userID).
		FirstOrCreate(like)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		// 新点赞，更新计数
		return r.db.WithContext(ctx).
			Model(&model.GameReview{}).
			Where("id = ?", reviewID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
	}
	return nil
}

func (r *reviewRepository) DeleteLike(ctx context.Context, reviewID, userID uint64) error {
	result := r.db.WithContext(ctx).
		Where("review_id = ? AND user_id = ?", reviewID, userID).
		Delete(&model.GameReviewLike{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return r.db.WithContext(ctx).
			Model(&model.GameReview{}).
			Where("id = ? AND like_count > 0", reviewID).
			UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error
	}
	return nil
}

func (r *reviewRepository) IsLiked(ctx context.Context, reviewID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.GameReviewLike{}).
		Where("review_id = ? AND user_id = ?", reviewID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *reviewRepository) ReplyReview(ctx context.Context, reviewID uint64, content string, repliedAt *time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.GameReview{}).
		Where("id = ?", reviewID).
		Updates(map[string]interface{}{
			"reply_content": content,
			"replied_at":    repliedAt,
		}).Error
}

func (r *reviewRepository) BatchIsLiked(ctx context.Context, reviewIDs []uint64, userID uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(reviewIDs))
	if len(reviewIDs) == 0 || userID == 0 {
		return result, nil
	}
	var likes []model.GameReviewLike
	if err := r.db.WithContext(ctx).
		Where("review_id IN ? AND user_id = ?", reviewIDs, userID).
		Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, l := range likes {
		result[l.ReviewID] = true
	}
	return result, nil
}
