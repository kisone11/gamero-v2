// Package service 提供社区系统的业务逻辑层。
// 本文件包含社区相关的所有业务逻辑：
//   - 话题管理（全量查询）
//   - 帖子发布、编辑、删除、详情、列表
//   - 帖子配图上传（云存储）
//   - 帖子互动（点赞/取消点赞、收藏/取消收藏）
//   - 帖子评论（发表、删除、树形列表）
//   - 评论点赞
//   - 内容举报
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/cache"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/metrics"
	"github.com/gamero/gamero/pkg/mq"
	"github.com/gamero/gamero/pkg/storage"
)

// 社区系统相关常量
const (
	// postMaxImages 每篇帖子最多 9 张配图
	postMaxImages = 9
	// postMaxVideos 每篇帖子最多 3 个视频
	postMaxVideos = 3
	// postMaxTopics 帖子最多关联 3 个话题
	postMaxTopics = 3
) // 合法的举报原因枚举
var validReportReasons = map[string]bool{
	"porn":      true,
	"spam":      true,
	"abuse":     true,
	"violation": true,
	"other":     true,
}

// ===========================
// 请求/响应数据结构
// ===========================

// CreatePostReq 发布帖子请求
type CreatePostReq struct {
	Title     string   `json:"title"`      // 帖子标题，最多 128 字符
	Content   string   `json:"content"`    // 帖子正文（Markdown），最多 50000 字符
	TopicIDs  []uint64 `json:"topic_ids"`  // 关联话题 ID 列表，最多 3 个
	Images    []string `json:"images"`     // 对象存储 key 列表，最多 9 张（可在创建时直接带上）
	ProjectID uint64   `json:"project_id"` // 关联项目 ID（0 表示不关联）
}

// UpdatePostReq 更新帖子请求
type UpdatePostReq struct {
	Title    *string  `json:"title"`     // 帖子标题
	Content  *string  `json:"content"`   // 帖子正文
	TopicIDs []uint64 `json:"topic_ids"` // 关联话题 ID 列表（覆盖更新，nil 表示不更新）
	Images   []string `json:"images"`    // 对象存储 key 列表（覆盖更新，nil 表示不更新）
}

// PostAuthorInfo 帖子作者基本信息（用于帖子列表和详情响应）
type PostAuthorInfo struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"` // 预签名 URL
}

// TopicInfo 话题简要信息（用于帖子列表和详情响应）
type TopicInfo struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// PostDetail 帖子详情响应（含作者信息、话题信息、配图预签名 URL、互动状态）
type PostDetail struct {
	ID           uint64          `json:"id"`
	AuthorID     uint64          `json:"author_id"`
	Title        string          `json:"title"`
	Content      string          `json:"content"`
	Topics       []*TopicInfo    `json:"topics"`     // 关联话题信息列表
	ImageURLs    []string        `json:"image_urls"` // 配图预签名 URL 列表（有效期 24h）
	VideoURLs    []string        `json:"video_urls"` // 视频公开 URL 列表
	ImageKeys    []string        `json:"image_keys"` // 配图对象存储 key 列表
	VideoKeys    []string        `json:"video_keys"` // 视频对象存储 key 列表
	ProjectID    uint64          `json:"project_id"`
	IsPinned     bool            `json:"is_pinned"`
	ViewCount    int             `json:"view_count"`
	LikeCount    int             `json:"like_count"`
	CommentCount int             `json:"comment_count"`
	CollectCount int             `json:"collect_count"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Author       *PostAuthorInfo `json:"author"`       // 作者信息
	IsLiked      bool            `json:"is_liked"`     // 当前用户是否已点赞
	IsCollected  bool            `json:"is_collected"` // 当前用户是否已收藏
}

// PostListItem 帖子列表项响应（比详情轻量，省略正文）
type PostListItem struct {
	ID             uint64          `json:"id"`
	AuthorID       uint64          `json:"author_id"`
	Title          string          `json:"title"`
	ContentSnippet string          `json:"content_snippet"` // 正文摘要（前 200 字符）
	Topics         []*TopicInfo    `json:"topics"`
	ImageURLs      []string        `json:"image_urls"`
	VideoURLs      []string        `json:"video_urls"`
	ImageKeys      []string        `json:"image_keys"`
	VideoKeys      []string        `json:"video_keys"`
	ProjectID      uint64          `json:"project_id"`
	IsPinned       bool            `json:"is_pinned"`
	ViewCount      int             `json:"view_count"`
	LikeCount      int             `json:"like_count"`
	CommentCount   int             `json:"comment_count"`
	CollectCount   int             `json:"collect_count"`
	CreatedAt      time.Time       `json:"created_at"`
	Author         *PostAuthorInfo `json:"author"`
	IsLiked        bool            `json:"is_liked"`
	IsCollected    bool            `json:"is_collected"`
}

// CreatePostCommentReq 发表帖子评论请求
type CreatePostCommentReq struct {
	Content   string `json:"content"`     // 评论内容，最多 1000 字符
	ReplyToID uint64 `json:"reply_to_id"` // 回复的评论 ID，一级评论传 0
}

// PostCommentWithReplies 带子评论的帖子评论结构（树形返回）
type PostCommentWithReplies struct {
	ID        uint64                    `json:"id"`
	PostID    uint64                    `json:"post_id"`
	AuthorID  uint64                    `json:"author_id"`
	Content   string                    `json:"content"`
	ReplyToID uint64                    `json:"reply_to_id"`
	LikeCount int                       `json:"like_count"`
	CreatedAt time.Time                 `json:"created_at"`
	Author    *PostAuthorInfo           `json:"author"`   // 评论作者信息
	IsLiked   bool                      `json:"is_liked"` // 当前用户是否已点赞
	Replies   []*PostCommentWithReplies `json:"replies"`  // 子评论列表
}

// CreateReportReq 提交举报请求
type CreateReportReq struct {
	TargetType string `json:"target_type"` // 目标类型：post / comment
	TargetID   uint64 `json:"target_id"`   // 目标 ID
	Reason     string `json:"reason"`      // 举报原因：porn/spam/abuse/violation/other
	Supplement string `json:"supplement"`  // 补充说明，最多 200 字
}

// UploadPostImageResult 帖子配图上传结果
type UploadPostImageResult struct {
	Key string `json:"key"` // 对象存储 key
	URL string `json:"url"` // 预签名访问 URL（有效期 24h）
}

// UploadPostVideoResult 帖子视频上传结果
type UploadPostVideoResult struct {
	Key      string `json:"key"`       // 对象存储 key
	URL      string `json:"url"`       // 预签名访问 URL（有效期 24h）
	MimeType string `json:"mime_type"` // 实际 MIME 类型
	Size     int64  `json:"size"`      // 文件大小（字节）
}

// ListPostsParams 帖子列表查询参数（service 层）
type ListPostsParams struct {
	TopicID        uint64 // 按话题 ID 筛选（0 表示不筛选）
	ProjectID      uint64 // 按关联项目 ID 筛选（0 表示不筛选）
	AuthorID       uint64 // 按作者 ID 筛选（0 表示不筛选）
	AuthorUsername string // 按作者用户名筛选（非空时内部解析为 AuthorID）
	Keyword        string // 关键词搜索（同时匹配 title 和 content）
	SortBy         string // 排序方式："hot"（按点赞数倒序）、""（默认按创建时间倒序）
	ViewerID       uint64 // 查看者 ID（0 表示未登录，用于查询是否已点赞/收藏）
	Page           int
	PageSize       int
}

// ===========================
// Service 接口定义
// ===========================

// CommunityService 社区系统业务接口
type CommunityService interface {
	// ===== 话题 =====

	// GetTopics 获取全量话题列表（不分页，一般不超过 100 个）
	GetTopics(ctx context.Context) ([]*model.Topic, error)
	// GetTopic 获取话题详情
	GetTopic(ctx context.Context, id uint64) (*model.Topic, error)

	// ===== 帖子 =====

	// CreatePost 发布帖子（验证话题存在并自动增加话题帖子计数）
	CreatePost(ctx context.Context, authorID uint64, req *CreatePostReq) (*PostDetail, error)
	// UpdatePost 更新帖子（只有作者可改）
	UpdatePost(ctx context.Context, authorID, postID uint64, req *UpdatePostReq) (*PostDetail, error)
	// DeletePost 软删除帖子（只有作者可删，同步减少话题帖子计数）
	DeletePost(ctx context.Context, authorID, postID uint64) error
	// GetPost 获取帖子详情（含作者信息、互动状态，浏览数 +1）
	GetPost(ctx context.Context, postID, viewerID uint64) (*PostDetail, error)
	// ListPosts 分页查询帖子列表（含作者信息、话题信息）
	ListPosts(ctx context.Context, params *ListPostsParams) ([]*PostListItem, int64, error)
	// ListHotPosts 查询热门帖子（按热度分数倒序）
	ListHotPosts(ctx context.Context, limit int) ([]*PostListItem, error)
	// GetUserPosts 查询用户发布的帖子（个人主页展示，分页）
	GetUserPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*PostListItem, int64, error)
	// SavePostImage 保存帖子图片 key（前端已直传后提交）
	SavePostImage(ctx context.Context, authorID, postID uint64, key string) (*UploadPostImageResult, error)
	// SavePostVideo 保存帖子视频 key（前端已直传后提交）
	SavePostVideo(ctx context.Context, authorID, postID uint64, key string) (*UploadPostVideoResult, error)
	// DeletePostVideo 删除帖子中的单个视频（通过 对象存储 key，仅作者可操作）
	DeletePostVideo(ctx context.Context, authorID, postID uint64, key string) error
	// DeletePostImage 删除帖子中的单个图片（通过 对象存储 key，仅作者可操作）
	DeletePostImage(ctx context.Context, authorID, postID uint64, key string) error

	// ===== 互动 =====

	// LikePost 点赞帖子（已点赞则幂等）
	LikePost(ctx context.Context, userID, postID uint64) error
	// UnlikePost 取消点赞帖子
	UnlikePost(ctx context.Context, userID, postID uint64) error
	// CollectPost 收藏帖子（已收藏则幂等）
	CollectPost(ctx context.Context, userID, postID uint64) error
	// UncollectPost 取消收藏帖子
	UncollectPost(ctx context.Context, userID, postID uint64) error

	// ===== 评论 =====

	// CreateComment 发表帖子评论（支持二级评论，replyToID=0 表示一级评论）
	CreateComment(ctx context.Context, authorID, postID uint64, req *CreatePostCommentReq) (*model.PostComment, error)
	// DeleteComment 软删除帖子评论（作者或帖子作者均可删）
	DeleteComment(ctx context.Context, operatorID, commentID uint64) error
	// ListComments 分页查询帖子评论（树形结构：一级评论 + replies 子列表）
	ListComments(ctx context.Context, postID, viewerID uint64, page, pageSize int, sortBy string) ([]*PostCommentWithReplies, int64, error)
	// LikeComment 帖子评论点赞（已点赞则幂等）
	LikeComment(ctx context.Context, userID, commentID uint64) error
	// UnlikeComment 取消帖子评论点赞
	UnlikeComment(ctx context.Context, userID, commentID uint64) error

	// ===== 举报 =====

	// CreateReport 提交内容举报（验证未重复举报）
	CreateReport(ctx context.Context, reporterID uint64, req *CreateReportReq) (*model.Report, error)
	// GetMyReports 查询用户自己的举报记录（分页）
	GetMyReports(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Report, int64, error)
	// RefreshHotScores 批量刷新最近 30 天帖子的热度分数
	RefreshHotScores(ctx context.Context) error
	// RefreshRecentHotScores 精准刷新最近 7 天内有互动（点赞/评论）的帖子热度分数（供 10 分钟定时任务调用）
	RefreshRecentHotScores(ctx context.Context) error
	// GetMyCollectedPosts 获取当前用户收藏的帖子列表（分页）
	GetMyCollectedPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*PostListItem, int64, error)
	// GetMyPosts 获取当前用户发布的帖子列表（分页）
	GetMyPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*PostListItem, int64, error)
}

// ===========================
// Service 实现
// ===========================

// communityService 社区 service 实现

type communityService struct {
	topicRepo       repository.TopicRepository
	postRepo        repository.PostRepository
	commentRepo     repository.PostCommentRepository
	postLikeRepo    repository.PostLikeRepository
	postCollectRepo repository.PostCollectRepository
	commentLikeRepo repository.PostCommentLikeRepository
	reportRepo      repository.ReportRepository
	userRepo        repository.UserRepository
	teamRepo        repository.TeamRepository
	followRepo      repository.FollowRepository
	*NotificationClient
	stor   *storage.Client
	broker mq.Broker
}

// NewCommunityService 创建 CommunityService 实例
func NewCommunityService(
	topicRepo repository.TopicRepository,
	postRepo repository.PostRepository,
	commentRepo repository.PostCommentRepository,
	postLikeRepo repository.PostLikeRepository,
	postCollectRepo repository.PostCollectRepository,
	commentLikeRepo repository.PostCommentLikeRepository,
	reportRepo repository.ReportRepository,
	userRepo repository.UserRepository,
	teamRepo repository.TeamRepository,
	followRepo repository.FollowRepository,
	stor *storage.Client,
	broker mq.Broker,
) CommunityService {
	svc := &communityService{
		topicRepo:          topicRepo,
		postRepo:           postRepo,
		commentRepo:        commentRepo,
		postLikeRepo:       postLikeRepo,
		postCollectRepo:    postCollectRepo,
		commentLikeRepo:    commentLikeRepo,
		reportRepo:         reportRepo,
		userRepo:           userRepo,
		teamRepo:           teamRepo,
		followRepo:         followRepo,
		stor:               stor,
		broker:             broker,
		NotificationClient: &NotificationClient{},
	}
	return svc
}

// sendNotification 写 DB 通知并实时 WS 推送（失败不阻断主流程）
func (s *communityService) sendNotification(ctx context.Context, n *model.Notification) {
	s.SendModel(ctx, n)
}

// ===========================
// 辅助函数
// ===========================

// buildPostAuthorInfo 从 User 模型构造帖子作者信息 DTO（含 avatar 预签名 URL）
func (s *communityService) presignURL(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	url, err := s.stor.PresignedGetURL(ctx, key, 24*time.Hour)
	if err != nil {
		return s.stor.GetPublicURL(key)
	}
	return url
}

func (s *communityService) buildPostAuthorInfo(ctx context.Context, user *model.User) *PostAuthorInfo {
	if user == nil {
		return nil
	}
	info := &PostAuthorInfo{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
	}
	// 生成 avatar 公开 URL
	if user.AvatarKey != "" {
		info.AvatarURL = s.presignURL(ctx, user.AvatarKey)
	}
	return info
}

// parseTopicIDs 将 JSON 字符串解析为 []uint64
func parseTopicIDs(topicIDsJSON string) []uint64 {
	var ids []uint64
	if topicIDsJSON != "" && topicIDsJSON != "[]" {
		if err := json.Unmarshal([]byte(topicIDsJSON), &ids); err != nil {
			logger.Warn("failed to unmarshal JSON", zap.Error(err))
		}
	}
	return ids
}

// buildTopicInfos 根据 ID 列表批量查询话题信息（避免 N+1）
func (s *communityService) buildTopicInfos(ctx context.Context, topicIDs []uint64) []*TopicInfo {
	if len(topicIDs) == 0 {
		return []*TopicInfo{}
	}
	topics, err := s.topicRepo.GetTopicsByIDs(ctx, topicIDs)
	if err != nil {
		return []*TopicInfo{}
	}
	infos := make([]*TopicInfo, 0, len(topics))
	for _, t := range topics {
		infos = append(infos, &TopicInfo{ID: t.ID, Name: t.Name})
	}
	return infos
}

// buildPostListItemsBatch 批量将帖子列表转换为 PostListItem（1 次 user 查询 + 1 次批量互动状态查询）
func (s *communityService) buildPostListItemsBatch(ctx context.Context, posts []*model.Post, viewerID uint64) []*PostListItem {
	if len(posts) == 0 {
		return []*PostListItem{}
	}

	// 批量收集 authorID
	authorIDs := make([]uint64, 0, len(posts))
	for _, p := range posts {
		authorIDs = append(authorIDs, p.AuthorID)
	}
	users, _ := s.userRepo.GetUsersByIDs(ctx, authorIDs)
	userMap := make(map[uint64]*model.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 批量获取互动状态
	likedSet := make(map[uint64]bool)
	collectedSet := make(map[uint64]bool)
	if viewerID > 0 {
		postIDs := make([]uint64, 0, len(posts))
		for _, p := range posts {
			postIDs = append(postIDs, p.ID)
		}
		if liked, err := s.postLikeRepo.GetLikedPostIDs(ctx, viewerID, postIDs); err == nil {
			for _, id := range liked {
				likedSet[id] = true
			}
		}
		if collected, err := s.postCollectRepo.GetCollectedPostIDs(ctx, viewerID, postIDs); err == nil {
			for _, id := range collected {
				collectedSet[id] = true
			}
		}
	}

	items := make([]*PostListItem, 0, len(posts))
	for _, post := range posts {
		topicIDs := parseTopicIDs(post.TopicIDs)
		contentRunes := []rune(post.Content)
		snippet := string(contentRunes)
		if len(contentRunes) > 200 {
			snippet = string(contentRunes[:200]) + "..."
		}
		item := &PostListItem{
			ID:             post.ID,
			AuthorID:       post.AuthorID,
			Title:          post.Title,
			ContentSnippet: snippet,
			Topics:         s.buildTopicInfos(ctx, topicIDs),
			ImageURLs:      buildImageURLs(ctx, post.Images),
			VideoURLs:      buildImageURLs(ctx, post.Videos),
			ImageKeys:      parseImageKeys(post.Images),
			VideoKeys:      parseImageKeys(post.Videos),
			IsPinned:       post.IsPinned,
			ViewCount:      post.ViewCount,
			LikeCount:      post.LikeCount,
			CommentCount:   post.CommentCount,
			CollectCount:   post.CollectCount,
			CreatedAt:      post.CreatedAt,
			IsLiked:        likedSet[post.ID],
			IsCollected:    collectedSet[post.ID],
		}
		if user, ok := userMap[post.AuthorID]; ok {
			item.Author = s.buildPostAuthorInfo(ctx, user)
		}
		items = append(items, item)
	}
	return items
}

// buildPostDetail 将帖子模型转换为详情 DTO
func (s *communityService) buildPostDetail(ctx context.Context, post *model.Post, viewerID uint64) *PostDetail {
	topicIDs := parseTopicIDs(post.TopicIDs)

	detail := &PostDetail{
		ID:           post.ID,
		AuthorID:     post.AuthorID,
		Title:        post.Title,
		Content:      post.Content,
		Topics:       s.buildTopicInfos(ctx, topicIDs),
		ImageURLs:    buildImageURLs(ctx, post.Images),
		VideoURLs:    buildImageURLs(ctx, post.Videos),
		ImageKeys:    parseImageKeys(post.Images),
		VideoKeys:    parseImageKeys(post.Videos),
		IsPinned:     post.IsPinned,
		ViewCount:    post.ViewCount,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
		CollectCount: post.CollectCount,
		CreatedAt:    post.CreatedAt,
		UpdatedAt:    post.UpdatedAt,
	}

	// 查询作者信息
	author, err := s.userRepo.GetUserByID(ctx, post.AuthorID)
	if err == nil {
		detail.Author = s.buildPostAuthorInfo(ctx, author)
	}

	// 查询当前用户互动状态
	if viewerID > 0 {
		liked, _ := s.postLikeRepo.HasPostLiked(ctx, post.ID, viewerID)
		collected, _ := s.postCollectRepo.HasPostCollected(ctx, post.ID, viewerID)
		detail.IsLiked = liked
		detail.IsCollected = collected
	}

	return detail
}

// buildPostListItem 将帖子模型转换为列表项 DTO（正文截断为摘要）
func (s *communityService) buildPostListItem(ctx context.Context, post *model.Post, viewerID uint64) *PostListItem {
	topicIDs := parseTopicIDs(post.TopicIDs)

	// 正文摘要：取前 200 个 rune 字符
	contentRunes := []rune(post.Content)
	snippet := string(contentRunes)
	if len(contentRunes) > 200 {
		snippet = string(contentRunes[:200]) + "..."
	}

	item := &PostListItem{
		ID:             post.ID,
		AuthorID:       post.AuthorID,
		Title:          post.Title,
		ContentSnippet: snippet,
		Topics:         s.buildTopicInfos(ctx, topicIDs),
		ImageURLs:      buildImageURLs(ctx, post.Images),
		VideoURLs:      buildImageURLs(ctx, post.Videos),
		IsPinned:       post.IsPinned,
		ViewCount:      post.ViewCount,
		LikeCount:      post.LikeCount,
		CommentCount:   post.CommentCount,
		CollectCount:   post.CollectCount,
		CreatedAt:      post.CreatedAt,
	}

	// 查询作者信息
	author, err := s.userRepo.GetUserByID(ctx, post.AuthorID)
	if err == nil {
		item.Author = s.buildPostAuthorInfo(ctx, author)
	}

	// 查询当前用户互动状态
	if viewerID > 0 {
		liked, _ := s.postLikeRepo.HasPostLiked(ctx, post.ID, viewerID)
		collected, _ := s.postCollectRepo.HasPostCollected(ctx, post.ID, viewerID)
		item.IsLiked = liked
		item.IsCollected = collected
	}

	return item
}

// ===========================
// 话题
// ===========================

// GetTopics 获取全量话题列表
func (s *communityService) GetTopics(ctx context.Context) ([]*model.Topic, error) {
	topics, err := s.topicRepo.ListTopics(ctx)
	if err != nil {
		return nil, err
	}
	return topics, nil
}

// GetTopic 获取话题详情
func (s *communityService) GetTopic(ctx context.Context, id uint64) (*model.Topic, error) {
	return s.topicRepo.GetTopicByID(ctx, id)
}

// ===========================
// 帖子
// ===========================

// validateCreatePostReq 校验发布帖子请求
func validateCreatePostReq(req *CreatePostReq) error {
	if strings.TrimSpace(req.Title) == "" {
		return apperrors.New(apperrors.CodeParamMissing, "帖子标题不能为空")
	}
	if len([]rune(req.Title)) > 128 {
		return apperrors.New(apperrors.CodeParamTooLong, "帖子标题最多 128 字符")
	}
	if strings.TrimSpace(req.Content) == "" {
		return apperrors.New(apperrors.CodeParamMissing, "帖子正文不能为空")
	}
	if len([]rune(req.Content)) > 50000 {
		return apperrors.New(apperrors.CodeParamTooLong, "帖子正文最多 50000 字符")
	}
	if len(req.TopicIDs) > postMaxTopics {
		return apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("关联话题最多 %d 个", postMaxTopics))
	}
	if len(req.Images) > postMaxImages {
		return apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("帖子配图最多 %d 张", postMaxImages))
	}
	return nil
}

// CreatePost 发布帖子
func (s *communityService) CreatePost(ctx context.Context, authorID uint64, req *CreatePostReq) (*PostDetail, error) {
	// 参数校验
	if err := validateCreatePostReq(req); err != nil {
		return nil, err
	}

	// 敏感词检测
	if filter.Contains(req.Title) || filter.Contains(req.Content) {
		return nil, apperrors.New(apperrors.CodeContentViolation, "内容包含敏感词，请修改后重试")
	}

	// 验证话题是否存在（去重）
	uniqueTopicIDs := deduplicateUint64(req.TopicIDs)
	for _, tid := range uniqueTopicIDs {
		if _, err := s.topicRepo.GetTopicByID(ctx, tid); err != nil {
			return nil, apperrors.Newf(apperrors.CodeNotFound, "话题 ID %d 不存在", tid)
		}
	}

	// 序列化话题 ID 列表
	topicIDsJSON := "[]"
	if len(uniqueTopicIDs) > 0 {
		b, err := json.Marshal(uniqueTopicIDs)
		if err != nil {
			return nil, fmt.Errorf("序列化话题 ID 列表失败: %w", err)
		}
		topicIDsJSON = string(b)
	}

	// 序列化图片 key 列表
	imagesJSON := "[]"
	if len(req.Images) > 0 {
		b, err := json.Marshal(req.Images)
		if err != nil {
			return nil, fmt.Errorf("序列化图片列表失败: %w", err)
		}
		imagesJSON = string(b)
	}

	post := &model.Post{
		AuthorID: authorID,
		Title:    strings.TrimSpace(req.Title),
		Content:  req.Content,
		TopicIDs: topicIDsJSON,
		Images:   imagesJSON,
	}

	if err := s.postRepo.CreatePost(ctx, post); err != nil {
		return nil, err
	}

	// 同步增加各话题的帖子计数
	for _, tid := range uniqueTopicIDs {
		if err := s.topicRepo.IncrPostCount(ctx, tid, 1); err != nil {
			logger.Warn("failed to update counts", zap.Error(err))
		}
	}

	// 异步发布帖子发布事件 → 触发粉丝 Feed 扩散
	if s.broker != nil {
		if ev, err := mq.NewEvent(mq.EventPostPublished, mq.PostPublishedPayload{
			PostID:   post.ID,
			AuthorID: authorID,
		}); err == nil {
			if err := s.broker.Publish(ctx, ev); err != nil {
				logger.Warn("failed to publish event", zap.Error(err))
			}
		}
	}

	// 异步向粉丝做 Feed 写扩散（仅对粉丝数 < FanoutThreshold 的普通用户）
	go func() {
		fanoutCtx := context.Background()
		followerIDs, err := s.followRepo.GetFollowerIDsUpToLimit(fanoutCtx, authorID, cache.FanoutThreshold)
		if err != nil || len(followerIDs) == 0 {
			return
		}
		feedFanout(fanoutCtx, "post", post.ID, float64(post.CreatedAt.Unix()), followerIDs)
	}()

	metrics.ContentCreatedTotal.WithLabelValues("post").Inc()

	return s.buildPostDetail(ctx, post, authorID), nil
}

// UpdatePost 更新帖子（只有作者可改）
func (s *communityService) UpdatePost(ctx context.Context, authorID, postID uint64, req *UpdatePostReq) (*PostDetail, error) {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	// 只有作者可编辑
	if post.AuthorID != authorID {
		return nil, apperrors.CodeError(apperrors.CodePostForbidden)
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, apperrors.New(apperrors.CodeParamMissing, "帖子标题不能为空")
		}
		if len([]rune(title)) > 128 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "帖子标题最多 128 字符")
		}
		if filter.Contains(title) {
			return nil, apperrors.New(apperrors.CodeContentViolation, "标题包含敏感词，请修改后重试")
		}
		updates["title"] = title
	}

	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			return nil, apperrors.New(apperrors.CodeParamMissing, "帖子正文不能为空")
		}
		if len([]rune(content)) > 50000 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "帖子正文最多 50000 字符")
		}
		if filter.Contains(content) {
			return nil, apperrors.New(apperrors.CodeContentViolation, "正文包含敏感词，请修改后重试")
		}
		updates["content"] = content
	}

	// 更新话题（带旧话题计数同步）
	if req.TopicIDs != nil {
		if len(req.TopicIDs) > postMaxTopics {
			return nil, apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("关联话题最多 %d 个", postMaxTopics))
		}
		uniqueTopicIDs := deduplicateUint64(req.TopicIDs)
		for _, tid := range uniqueTopicIDs {
			if _, err := s.topicRepo.GetTopicByID(ctx, tid); err != nil {
				return nil, apperrors.Newf(apperrors.CodeNotFound, "话题 ID %d 不存在", tid)
			}
		}

		// 减少旧话题计数
		oldTopicIDs := parseTopicIDs(post.TopicIDs)
		for _, oldID := range oldTopicIDs {
			if err := s.topicRepo.IncrPostCount(ctx, oldID, -1); err != nil {
				logger.Warn("failed to update counts", zap.Error(err))
			}
		}

		// 增加新话题计数
		for _, newID := range uniqueTopicIDs {
			if err := s.topicRepo.IncrPostCount(ctx, newID, 1); err != nil {
				logger.Warn("failed to update counts", zap.Error(err))
			}
		}

		b, err := json.Marshal(uniqueTopicIDs)
		if err != nil {
			return nil, fmt.Errorf("序列化话题 ID 列表失败: %w", err)
		}
		updates["topic_ids"] = string(b)
	}

	if req.Images != nil {
		if len(req.Images) > postMaxImages {
			return nil, apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("帖子配图最多 %d 张", postMaxImages))
		}
		b, err := json.Marshal(req.Images)
		if err != nil {
			return nil, fmt.Errorf("序列化图片列表失败: %w", err)
		}
		updates["images"] = string(b)
	}

	if len(updates) == 0 {
		return s.buildPostDetail(ctx, post, authorID), nil
	}

	if err := s.postRepo.UpdatePost(ctx, postID, updates); err != nil {
		return nil, err
	}

	// 清除帖子详情缓存
	if err := cache.Del(ctx, cache.PostDetailKey(postID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}

	// 重新查询最新数据
	updated, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	return s.buildPostDetail(ctx, updated, authorID), nil
}

// DeletePost 软删除帖子（只有作者可删）
func (s *communityService) DeletePost(ctx context.Context, authorID, postID uint64) error {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return err
	}

	if post.AuthorID != authorID {
		return apperrors.CodeError(apperrors.CodePostForbidden)
	}

	if err := s.postRepo.DeletePost(ctx, postID); err != nil {
		return err
	}

	// 清除帖子详情缓存和热榜缓存
	if err := cache.Del(ctx, cache.PostDetailKey(postID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}

	// 同步减少关联话题的帖子计数
	topicIDs := parseTopicIDs(post.TopicIDs)
	for _, tid := range topicIDs {
		if err := s.topicRepo.IncrPostCount(ctx, tid, -1); err != nil {
			logger.Warn("failed to update counts", zap.Error(err))
		}
	}

	return nil
}

// GetPost 获取帖子详情（UV 去重浏览数 +1）
func (s *communityService) GetPost(ctx context.Context, postID, viewerID uint64) (*PostDetail, error) {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	// UV 去重：同一访客（userID 或 guest:IP 由上层传入）当天只计一次浏览
	// viewerID=0 表示匿名用户，仍然计入 HLL（以 "anon" 作为标识占位，精度略低）
	go func() {
		hllKey := cache.PostViewHLLKey(postID, time.Now().Format("2006-01-02"))
		member := fmt.Sprintf("%d", viewerID) // 0 = 匿名
		isNew, _ := cache.PFAddAndCheckNew(context.Background(), hllKey, member, cache.TTLViewHLL)
		if isNew {
			if err := s.postRepo.IncrViewCount(context.Background(), postID); err != nil {
				logger.Warn("failed to update counts", zap.Error(err))
			}
		}
	}()

	return s.buildPostDetail(ctx, post, viewerID), nil
}

// ListPosts 分页查询帖子列表
func (s *communityService) ListPosts(ctx context.Context, params *ListPostsParams) ([]*PostListItem, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	// 若传入 AuthorUsername，内部解析为 AuthorID（避免 handler 层直接依赖 repository）
	if params.AuthorUsername != "" && params.AuthorID == 0 {
		user, err := s.userRepo.GetUserByUsername(ctx, params.AuthorUsername)
		if err != nil {
			return nil, 0, err
		}
		params.AuthorID = user.ID
	}

	repoParams := &repository.ListPostsParams{
		TopicID:   params.TopicID,
		ProjectID: params.ProjectID,
		AuthorID:  params.AuthorID,
		Keyword:   params.Keyword,
		SortBy:    params.SortBy,
		Offset:    (params.Page - 1) * params.PageSize,
		Limit:     params.PageSize,
	}

	posts, total, err := s.postRepo.ListPosts(ctx, repoParams)
	if err != nil {
		return nil, 0, err
	}

	items := s.buildPostListItemsBatch(ctx, posts, params.ViewerID)
	return items, total, nil
}

// ListHotPosts 查询热门帖子（按点赞数倒序）
func (s *communityService) ListHotPosts(ctx context.Context, limit int) ([]*PostListItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// ===== 读缓存 =====
	cacheKey := cache.PostHotListKey(limit)
	var cachedItems []*PostListItem
	if hit, _ := cache.GetJSON(ctx, cacheKey, &cachedItems); hit {
		return cachedItems, nil
	}

	posts, err := s.postRepo.ListHotPosts(ctx, limit)
	if err != nil {
		return nil, err
	}
	items := s.buildPostListItemsBatch(ctx, posts, 0)

	// ===== 写缓存 =====
	if err := cache.SetJSON(ctx, cacheKey, items, cache.TTLHotList); err != nil {
		logger.Warn("failed to set cache", zap.Error(err))
	}

	return items, nil
}

// GetUserPosts 查询用户发布的帖子（分页）
func (s *communityService) GetUserPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*PostListItem, int64, error) {
	return s.ListPosts(ctx, &ListPostsParams{
		AuthorID: userID,
		Page:     page,
		PageSize: pageSize,
	})
}

// SavePostImage 保存帖子图片 key（前端已通过预签名 URL 直传后提交）
func (s *communityService) SavePostImage(ctx context.Context, authorID, postID uint64, key string) (*UploadPostImageResult, error) {
	if key == "" {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "key 不能为空")
	}
	// 验证作者权限
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post.AuthorID != authorID {
		return nil, apperrors.CodeError(apperrors.CodePostForbidden)
	}
	// 检查图片数量上限
	var existingKeys []string
	if post.Images != "" && post.Images != "[]" {
		if err := json.Unmarshal([]byte(post.Images), &existingKeys); err != nil {
			logger.Warn("failed to unmarshal JSON", zap.Error(err))
		}
	}
	if len(existingKeys) >= postMaxImages {
		return nil, apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("帖子配图数量已达上限（最多 %d 张）", postMaxImages))
	}
	// 追加 key
	existingKeys = append(existingKeys, key)
	newImagesJSON, _ := json.Marshal(existingKeys)
	if err := s.postRepo.UpdatePost(ctx, postID, map[string]interface{}{"images": string(newImagesJSON)}); err != nil {
		return nil, err
	}
	// 生成预签名 GET URL
	url, _ := s.stor.PresignedGetURL(context.Background(), key, 24*time.Hour)
	return &UploadPostImageResult{Key: key, URL: url}, nil
}

// SavePostVideo 保存帖子视频 key（前端已通过预签名 URL 直传后提交）
func (s *communityService) SavePostVideo(ctx context.Context, authorID, postID uint64, key string) (*UploadPostVideoResult, error) {
	if key == "" {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "key 不能为空")
	}
	// 验证作者权限
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post.AuthorID != authorID {
		return nil, apperrors.CodeError(apperrors.CodeForbidden)
	}
	// 检查视频数量上限
	var existingVideos []string
	if post.Videos != "" && post.Videos != "[]" {
		if err := json.Unmarshal([]byte(post.Videos), &existingVideos); err != nil {
			logger.Warn("failed to unmarshal JSON", zap.Error(err))
		}
	}
	if len(existingVideos) >= postMaxVideos {
		return nil, apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("帖子视频数量已达上限（最多 %d 个）", postMaxVideos))
	}
	// 追加 key
	existingVideos = append(existingVideos, key)
	newVideosJSON, _ := json.Marshal(existingVideos)
	if err := s.postRepo.UpdatePost(ctx, postID, map[string]interface{}{"videos": string(newVideosJSON)}); err != nil {
		return nil, err
	}
	// 生成预签名 GET URL
	url, _ := s.stor.PresignedGetURL(context.Background(), key, 24*time.Hour)
	return &UploadPostVideoResult{Key: key, URL: url}, nil
}

// DeletePostVideo 删除帖子中的单个视频（通过 对象存储 key，仅作者可操作）
func (s *communityService) DeletePostVideo(ctx context.Context, authorID, postID uint64, key string) error {
	if key == "" {
		return apperrors.New(apperrors.CodeParamInvalid, "video key 不能为空")
	}
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return err
	}
	if post.AuthorID != authorID {
		return apperrors.CodeError(apperrors.CodeForbidden)
	}

	var videos []string
	if post.Videos != "" && post.Videos != "[]" {
		if err := json.Unmarshal([]byte(post.Videos), &videos); err != nil {
			logger.Warn("failed to unmarshal JSON", zap.Error(err))
		}
	}

	found := false
	newVideos := make([]string, 0, len(videos))
	for _, v := range videos {
		if v == key {
			found = true
		} else {
			newVideos = append(newVideos, v)
		}
	}
	if !found {
		return apperrors.New(apperrors.CodeNotFound, "视频不存在或不属于该帖子")
	}

	if err := s.stor.Delete(ctx, key); err != nil {
		logger.Warn("failed to delete storage object", zap.Error(err))
	}

	newVideosJSON, _ := json.Marshal(newVideos)
	return s.postRepo.UpdatePost(ctx, postID, map[string]interface{}{"videos": string(newVideosJSON)})
}

// DeletePostImage 删除帖子中的单个图片（通过 对象存储 key，仅作者可操作）
func (s *communityService) DeletePostImage(ctx context.Context, authorID, postID uint64, key string) error {
	if key == "" {
		return apperrors.New(apperrors.CodeParamInvalid, "image key 不能为空")
	}
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return err
	}
	if post.AuthorID != authorID {
		return apperrors.CodeError(apperrors.CodeForbidden)
	}

	var images []string
	if post.Images != "" && post.Images != "[]" {
		if err := json.Unmarshal([]byte(post.Images), &images); err != nil {
			logger.Warn("failed to unmarshal JSON", zap.Error(err))
		}
	}

	found := false
	newImages := make([]string, 0, len(images))
	for _, img := range images {
		if img == key {
			found = true
		} else {
			newImages = append(newImages, img)
		}
	}
	if !found {
		return apperrors.New(apperrors.CodeNotFound, "图片不存在或不属于该帖子")
	}

	if err := s.stor.Delete(ctx, key); err != nil {
		logger.Warn("failed to delete storage object", zap.Error(err))
	}

	newImagesJSON, _ := json.Marshal(newImages)
	return s.postRepo.UpdatePost(ctx, postID, map[string]interface{}{"images": string(newImagesJSON)})
}

// ===========================
// 互动（点赞/取消点赞、收藏/取消收藏）
// ===========================

// LikePost 点赞帖子（已点赞则幂等）
func (s *communityService) LikePost(ctx context.Context, userID, postID uint64) error {
	// 验证帖子存在
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return err
	}

	// 拉黑检查：不能对拉黑关系中的用户内容操作
	// 已点赞则幂等
	hasLiked, err := s.postLikeRepo.HasPostLiked(ctx, postID, userID)
	if err != nil {
		return fmt.Errorf("查询点赞状态失败: %w", err)
	}
	if hasLiked {
		return nil
	}

	if err := s.postLikeRepo.CreatePostLike(ctx, &model.PostLike{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return err
	}

	// 同步更新帖子点赞数冗余字段（+1）
	if err := s.postRepo.UpdateCounts(ctx, postID, "like_count", 1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}

	// 异步通知帖子作者（不通知自己）
	if post.AuthorID != userID {
		go func() {
			liker, err := s.userRepo.GetUserByID(ctx, userID)
			if err != nil {
				return
			}
			name := liker.Nickname
			if name == "" {
				name = liker.Username
			}
			title := post.Title
			if len([]rune(title)) > 20 {
				title = string([]rune(title)[:20]) + "..."
			}
			s.sendNotification(ctx, &model.Notification{
				UserID:   post.AuthorID,
				Type:     model.NotificationTypePostLiked,
				Title:    "有人赞了你的帖子",
				Content:  fmt.Sprintf("%s 赞了你的帖子「%s」", name, title),
				Metadata: fmt.Sprintf(`{"post_id":%d,"liker_id":%d}`, postID, userID),
			})
		}()
	}
	return nil
}

// UnlikePost 取消点赞帖子
func (s *communityService) UnlikePost(ctx context.Context, userID, postID uint64) error {
	// 验证帖子存在
	if _, err := s.postRepo.GetPostByID(ctx, postID); err != nil {
		return err
	}

	if err := s.postLikeRepo.DeletePostLike(ctx, postID, userID); err != nil {
		return err
	}

	// 同步更新帖子点赞数冗余字段（-1）
	if err := s.postRepo.UpdateCounts(ctx, postID, "like_count", -1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// CollectPost 收藏帖子（已收藏则幂等）
func (s *communityService) CollectPost(ctx context.Context, userID, postID uint64) error {
	// 验证帖子存在
	if _, err := s.postRepo.GetPostByID(ctx, postID); err != nil {
		return err
	}

	// 已收藏则幂等
	hasCollected, err := s.postCollectRepo.HasPostCollected(ctx, postID, userID)
	if err != nil {
		return fmt.Errorf("查询收藏状态失败: %w", err)
	}
	if hasCollected {
		return nil
	}

	if err := s.postCollectRepo.CreatePostCollect(ctx, &model.PostCollect{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return err
	}

	// 同步更新帖子收藏数冗余字段（+1）
	if err := s.postRepo.UpdateCounts(ctx, postID, "collect_count", 1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// UncollectPost 取消收藏帖子
func (s *communityService) UncollectPost(ctx context.Context, userID, postID uint64) error {
	// 验证帖子存在
	if _, err := s.postRepo.GetPostByID(ctx, postID); err != nil {
		return err
	}

	if err := s.postCollectRepo.DeletePostCollect(ctx, postID, userID); err != nil {
		return err
	}

	// 同步更新帖子收藏数冗余字段（-1）
	if err := s.postRepo.UpdateCounts(ctx, postID, "collect_count", -1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// ===========================
// 评论
// ===========================

// CreateComment 发表帖子评论
func (s *communityService) CreateComment(ctx context.Context, authorID, postID uint64, req *CreatePostCommentReq) (*model.PostComment, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, apperrors.New(apperrors.CodeParamMissing, "评论内容不能为空")
	}
	if len([]rune(content)) > 1000 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "评论内容最多 1000 字符")
	}
	if filter.Contains(content) {
		return nil, apperrors.New(apperrors.CodeContentViolation, "评论包含敏感词，请修改后重试")
	}

	// 验证帖子存在
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	// 如果是子评论，验证父评论存在且属于同一帖子
	if req.ReplyToID > 0 {
		parent, err := s.commentRepo.GetCommentByID(ctx, req.ReplyToID)
		if err != nil {
			return nil, apperrors.New(apperrors.CodeCommentNotFound, "被回复的评论不存在")
		}
		if parent.PostID != postID {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "被回复的评论不属于此帖子")
		}
		// 只支持二级评论，不允许回复子评论
		if parent.ReplyToID != 0 {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "仅支持二级评论，不能回复子评论")
		}
	}

	comment := &model.PostComment{
		PostID:    postID,
		AuthorID:  authorID,
		Content:   content,
		ReplyToID: req.ReplyToID,
	}

	if err := s.commentRepo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	// 同步更新帖子评论数冗余字段（+1）
	if err := s.postRepo.UpdateCounts(ctx, postID, "comment_count", 1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}

	// 异步通知帖子作者（不通知自己）
	if post.AuthorID != authorID {
		go func() {
			commenter, err := s.userRepo.GetUserByID(ctx, authorID)
			if err != nil {
				return
			}
			name := commenter.Nickname
			if name == "" {
				name = commenter.Username
			}
			// 评论内容截断预览（最多30字）
			preview := []rune(content)
			suffix := ""
			if len(preview) > 30 {
				preview = preview[:30]
				suffix = "..."
			}
			s.sendNotification(ctx, &model.Notification{
				UserID:   post.AuthorID,
				Type:     model.NotificationTypePostCommented,
				Title:    "有人评论了你的帖子",
				Content:  fmt.Sprintf("%s 评论了你的帖子：%s%s", name, string(preview), suffix),
				Metadata: fmt.Sprintf(`{"post_id":%d,"comment_id":%d,"commenter_id":%d}`, postID, comment.ID, authorID),
			})
		}()
	}

	// 异步通过消息队列发布评论创建事件（触发 @提及通知等），不阻塞主流程
	if s.broker != nil {
		event, err := mq.NewEvent(mq.EventCommentCreated, mq.CommentCreatedPayload{
			CommentID: comment.ID,
			PostID:    postID,
			LogID:     0,
			AuthorID:  comment.AuthorID,
			Content:   comment.Content,
		})
		if err == nil {
			if err := s.broker.Publish(ctx, event); err != nil {
				logger.Warn("failed to publish event", zap.Error(err))
			}
		}
	}

	return comment, nil
}

// sendMentionNotifications 解析评论内容中的 @username，向被提及用户发送通知（异步调用）
func (s *communityService) sendMentionNotifications(ctx context.Context, authorID uint64, content string, commentID uint64) {
	usernames := extractMentions(content)
	for _, username := range usernames {
		user, err := s.userRepo.GetUserByUsername(ctx, username)
		if err != nil || user.ID == authorID {
			continue
		}
		s.Send(ctx, &SendNotificationReq{
			UserID:   user.ID,
			Type:     model.NotificationTypeMention,
			Title:    "有人在评论中提到了你",
			Content:  fmt.Sprintf("@%s 在评论中提到了你", username),
			Metadata: map[string]interface{}{"comment_id": commentID},
		})
	}
}

// DeleteComment 软删除帖子评论（作者或帖子作者均可删）
func (s *communityService) DeleteComment(ctx context.Context, operatorID, commentID uint64) error {
	comment, err := s.commentRepo.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	// 查询帖子（获取帖子作者）
	post, err := s.postRepo.GetPostByID(ctx, comment.PostID)
	if err != nil {
		return err
	}

	// 只有评论者或帖子作者可删
	if comment.AuthorID != operatorID && post.AuthorID != operatorID {
		return apperrors.CodeError(apperrors.CodeCommentForbidden)
	}

	if err := s.commentRepo.DeleteComment(ctx, commentID); err != nil {
		return err
	}

	// 同步更新帖子评论数冗余字段（-1）
	if err := s.postRepo.UpdateCounts(ctx, comment.PostID, "comment_count", -1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// ListComments 分页查询帖子评论（树形结构）
func (s *communityService) ListComments(ctx context.Context, postID, viewerID uint64, page, pageSize int, sortBy string) ([]*PostCommentWithReplies, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询一级评论（reply_to_id=0）
	topComments, total, err := s.commentRepo.ListCommentsByPostID(ctx, postID, offset, pageSize, sortBy)
	if err != nil {
		return nil, 0, err
	}

	if len(topComments) == 0 {
		return []*PostCommentWithReplies{}, total, nil
	}

	// 收集一级评论 ID，用于批量查询子评论
	parentIDs := make([]uint64, 0, len(topComments))
	for _, c := range topComments {
		parentIDs = append(parentIDs, c.ID)
	}

	// 批量查询子评论
	childComments, err := s.commentRepo.ListChildCommentsByParentIDs(ctx, postID, parentIDs)
	if err != nil {
		return nil, 0, err
	}

	// 批量收集所有评论的 authorID（一级 + 子评论）
	allAuthorIDs := make([]uint64, 0, len(topComments)+len(childComments))
	for _, c := range topComments {
		allAuthorIDs = append(allAuthorIDs, c.AuthorID)
	}
	for _, c := range childComments {
		allAuthorIDs = append(allAuthorIDs, c.AuthorID)
	}
	authorUsers, _ := s.userRepo.GetUsersByIDs(ctx, allAuthorIDs)
	authorMap := make(map[uint64]*model.User, len(authorUsers))
	for _, u := range authorUsers {
		authorMap[u.ID] = u
	}

	// 构建父评论 ID → 子评论列表的映射
	// 批量查询评论点赞状态（避免 N+1）
	var likedCommentMap map[uint64]bool
	if viewerID > 0 {
		allCommentIDs := make([]uint64, 0, len(topComments)+len(childComments))
		for _, c := range topComments {
			allCommentIDs = append(allCommentIDs, c.ID)
		}
		for _, c := range childComments {
			allCommentIDs = append(allCommentIDs, c.ID)
		}
		likedCommentMap, _ = s.commentLikeRepo.GetLikedPostCommentIDs(ctx, viewerID, allCommentIDs)
	}

	// 构建父评论 ID → 子评论列表的映射
	childMap := make(map[uint64][]*PostCommentWithReplies)
	for _, child := range childComments {
		childMap[child.ReplyToID] = append(childMap[child.ReplyToID], &PostCommentWithReplies{
			ID:        child.ID,
			PostID:    child.PostID,
			AuthorID:  child.AuthorID,
			Content:   child.Content,
			ReplyToID: child.ReplyToID,
			LikeCount: child.LikeCount,
			CreatedAt: child.CreatedAt,
			Author:    s.buildPostAuthorInfo(ctx, authorMap[child.AuthorID]),
			IsLiked:   likedCommentMap[child.ID],
			Replies:   []*PostCommentWithReplies{},
		})
	}

	// 组装树形结构
	result := make([]*PostCommentWithReplies, 0, len(topComments))
	for _, c := range topComments {
		replies := childMap[c.ID]
		if replies == nil {
			replies = []*PostCommentWithReplies{}
		}
		result = append(result, &PostCommentWithReplies{
			ID:        c.ID,
			PostID:    c.PostID,
			AuthorID:  c.AuthorID,
			Content:   c.Content,
			ReplyToID: c.ReplyToID,
			LikeCount: c.LikeCount,
			CreatedAt: c.CreatedAt,
			Author:    s.buildPostAuthorInfo(ctx, authorMap[c.AuthorID]),
			IsLiked:   likedCommentMap[c.ID],
			Replies:   replies,
		})
	}

	return result, total, nil
}

// LikeComment 帖子评论点赞（已点赞则幂等）
func (s *communityService) LikeComment(ctx context.Context, userID, commentID uint64) error {
	// 验证评论存在
	if _, err := s.commentRepo.GetCommentByID(ctx, commentID); err != nil {
		return err
	}

	// 已点赞则幂等
	hasLiked, err := s.commentLikeRepo.HasPostCommentLiked(ctx, commentID, userID)
	if err != nil {
		return fmt.Errorf("查询评论点赞状态失败: %w", err)
	}
	if hasLiked {
		return nil
	}

	if err := s.commentLikeRepo.CreatePostCommentLike(ctx, &model.PostCommentLike{
		CommentID: commentID,
		UserID:    userID,
	}); err != nil {
		return err
	}

	// 同步更新评论点赞数冗余字段（+1）
	if err := s.commentRepo.IncrCommentLikeCount(ctx, commentID, 1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// UnlikeComment 取消帖子评论点赞
func (s *communityService) UnlikeComment(ctx context.Context, userID, commentID uint64) error {
	// 验证评论存在
	if _, err := s.commentRepo.GetCommentByID(ctx, commentID); err != nil {
		return err
	}

	if err := s.commentLikeRepo.DeletePostCommentLike(ctx, commentID, userID); err != nil {
		return err
	}

	// 同步更新评论点赞数冗余字段（-1）
	if err := s.commentRepo.IncrCommentLikeCount(ctx, commentID, -1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// ===========================
// 举报
// ===========================

// CreateReport 提交内容举报
func (s *communityService) CreateReport(ctx context.Context, reporterID uint64, req *CreateReportReq) (*model.Report, error) {
	// 验证目标类型
	validTargetTypes := map[string]bool{
		"post": true, "comment": true,
		"game_review": true, "project_release": true,
		"project": true,
	}
	if !validTargetTypes[req.TargetType] {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "目标类型无效，支持: post / comment / game_review / project_release / project")
	}

	// 验证目标 ID 非零
	if req.TargetID == 0 {
		return nil, apperrors.New(apperrors.CodeParamMissing, "目标 ID 不能为空")
	}

	// 验证目标内容是否存在（post/comment 强校验，新类型仅做 ID 非零校验）
	switch req.TargetType {
	case "post":
		if _, err := s.postRepo.GetPostByID(ctx, req.TargetID); err != nil {
			return nil, apperrors.CodeError(apperrors.CodePostNotFound)
		}
	case "comment":
		if _, err := s.commentRepo.GetCommentByID(ctx, req.TargetID); err != nil {
			return nil, apperrors.CodeError(apperrors.CodeCommentNotFound)
		}
		// game_review / project_release：ID 非零即可，不做额外数据库校验
	}

	// 验证举报原因
	if !validReportReasons[req.Reason] {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "举报原因无效，支持：porn/spam/abuse/violation/other")
	}

	// 补充说明长度校验
	if len([]rune(req.Supplement)) > 200 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "补充说明最多 200 字")
	}

	// 验证未重复举报
	hasReported, err := s.reportRepo.HasReported(ctx, reporterID, req.TargetType, req.TargetID)
	if err != nil {
		return nil, fmt.Errorf("查询举报状态失败: %w", err)
	}
	if hasReported {
		return nil, apperrors.New(apperrors.CodeConflict, "已举报过该内容，不可重复举报")
	}

	report := &model.Report{
		ReporterID: reporterID,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Reason:     req.Reason,
		Supplement: strings.TrimSpace(req.Supplement),
		Status:     "pending",
	}

	if err := s.reportRepo.CreateReport(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

// GetMyReports 查询用户自己的举报记录（分页）
func (s *communityService) GetMyReports(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Report, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.reportRepo.ListReportsByReporterID(ctx, userID, offset, pageSize)
}

// RefreshHotScores 批量刷新最近 30 天帖子的热度分数
func (s *communityService) RefreshHotScores(ctx context.Context) error {
	posts, err := s.postRepo.ListPostsForHotUpdate(ctx, 1000)
	if err != nil {
		return err
	}
	updates := make(map[uint64]float64, len(posts))
	for _, p := range posts {
		updates[p.ID] = CalcPostHotScore(p)
	}
	if err := s.postRepo.BatchUpdateHotScores(ctx, updates); err != nil {
		return err
	}

	// ===== 清除热榜缓存（常用 limit 挡位） =====
	for _, limit := range []int{10, 20, 50, 100} {
		if err := cache.Del(ctx, cache.PostHotListKey(limit)); err != nil {
			logger.Warn("failed to delete cache", zap.Error(err))
		}
	}

	return nil
}

// RefreshRecentHotScores 精准刷新最近 7 天内有互动（点赞/评论）的帖子热度分数。
// 相比 RefreshHotScores（30天全量），本方法仅更新活跃帖子，适合高频（10分钟）调度。
func (s *communityService) RefreshRecentHotScores(ctx context.Context) error {
	// 查询最近 7 天内有互动的帖子，上限 2000 条
	posts, err := s.postRepo.ListActivePostsWithinDays(ctx, 7, 2000)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		return nil
	}
	updates := make(map[uint64]float64, len(posts))
	for _, p := range posts {
		updates[p.ID] = CalcPostHotScore(p)
	}
	if err := s.postRepo.BatchUpdateHotScores(ctx, updates); err != nil {
		return err
	}
	// 清除热榜缓存
	for _, limit := range []int{10, 20, 50, 100} {
		if err := cache.Del(ctx, cache.PostHotListKey(limit)); err != nil {
			logger.Warn("failed to delete cache", zap.Error(err))
		}
	}
	return nil
}

// ===========================
// 通用辅助函数
// ===========================

// deduplicateUint64 对 uint64 切片去重（保持原顺序）
func deduplicateUint64(ids []uint64) []uint64 {
	seen := make(map[uint64]bool)
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

// GetMyCollectedPosts 获取当前用户收藏的帖子列表（分页）
func (s *communityService) GetMyCollectedPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*PostListItem, int64, error) {
	offset := (page - 1) * pageSize
	postIDs, total, err := s.postCollectRepo.ListCollectedPostsByUserID(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if len(postIDs) == 0 {
		return []*PostListItem{}, total, nil
	}

	posts, err := s.postRepo.GetPostsByIDs(ctx, postIDs)
	if err != nil {
		return nil, 0, err
	}

	items := s.buildPostListItemsBatch(ctx, posts, userID)
	return items, total, nil
}

// GetMyPosts 获取当前用户发布的帖子列表（分页），代理到 GetUserPosts
func (s *communityService) GetMyPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*PostListItem, int64, error) {
	return s.GetUserPosts(ctx, userID, page, pageSize)
}
