// Package repository 提供社区系统的数据库访问层。
// 本文件包含社区相关的所有仓储接口和实现：
//   - TopicRepository: 话题 CRUD
//   - PostRepository: 帖子 CRUD（含筛选/分页/排序）
//   - PostCommentRepository: 帖子评论 CRUD（含树形查询）
//   - PostLikeRepository: 帖子点赞
//   - PostCollectRepository: 帖子收藏
//   - PostCommentLikeRepository: 帖子评论点赞
//   - ReportRepository: 内容举报
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gamero/gamero/internal/model"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"gorm.io/gorm"
)

// ===========================
// 通用查询参数
// ===========================

// ListPostsParams 帖子列表查询参数
type ListPostsParams struct {
	TopicID   uint64 // 按话题 ID 筛选（0 表示不筛选）
	ProjectID uint64 // 按项目 ID 筛选（0 表示不筛选）
	AuthorID  uint64 // 按作者 ID 筛选（0 表示不筛选）
	Keyword   string // 关键词搜索（同时匹配 title 和 content）
	SortBy    string // 排序方式："hot"（按热度分数倒序）、""（默认按创建时间倒序）
	Offset    int
	Limit     int
}

// ===========================
// 接口定义
// ===========================

// TopicRepository 话题仓储接口
type TopicRepository interface {
	// ListTopics 获取全量话题列表
	ListTopics(ctx context.Context) ([]*model.Topic, error)
	// GetTopicByID 根据 ID 查询话题
	GetTopicByID(ctx context.Context, id uint64) (*model.Topic, error)
	// GetTopicsByIDs 按 ID 列表批量查询话题
	GetTopicsByIDs(ctx context.Context, ids []uint64) ([]*model.Topic, error)
	// IncrPostCount 原子增减话题的帖子计数（delta 为 +1 或 -1）
	IncrPostCount(ctx context.Context, id uint64, delta int) error
}

// PostRepository 帖子仓储接口
type PostRepository interface {
	// CreatePost 创建帖子，返回含 ID 的帖子对象
	CreatePost(ctx context.Context, post *model.Post) error
	// GetPostByID 根据 ID 查询帖子（未软删除）
	GetPostByID(ctx context.Context, id uint64) (*model.Post, error)
	// GetPostsByIDs 按 ID 列表批量获取帖子（跳过软删除条目）
	GetPostsByIDs(ctx context.Context, ids []uint64) ([]*model.Post, error)
	// UpdatePost 更新帖子指定字段
	UpdatePost(ctx context.Context, id uint64, updates map[string]interface{}) error
	// DeletePost 软删除帖子
	DeletePost(ctx context.Context, id uint64) error
	// ListPosts 分页查询帖子列表（支持话题/项目/作者筛选、关键词搜索、排序）
	ListPosts(ctx context.Context, params *ListPostsParams) ([]*model.Post, int64, error)
	// ListHotPosts 查询热门帖子（按热度分数倒序）
	ListHotPosts(ctx context.Context, limit int) ([]*model.Post, error)
	// UpdateCounts 原子增减帖子冗余数字段（like_count / comment_count / collect_count）
	UpdateCounts(ctx context.Context, id uint64, field string, delta int) error
	// IncrViewCount 浏览数 +1（原子操作）
	IncrViewCount(ctx context.Context, id uint64) error
	// ListPostsForHotUpdate 查询需要刷新热度分数的帖子列表（最近 30 天）
	ListPostsForHotUpdate(ctx context.Context, limit int) ([]*model.Post, error)
	// BatchUpdateHotScores 批量更新帖子热度分数
	BatchUpdateHotScores(ctx context.Context, updates map[uint64]float64) error
	// ListActivePostsWithinDays 查询最近 N 天内有互动（点赞/评论）的帖子
	ListActivePostsWithinDays(ctx context.Context, days int, limit int) ([]*model.Post, error)
	// ListPublicPostsByAuthorIDs 批量查询多个作者的公开帖子（用于动态流）
	// 查询 author_id IN authorIDs 且未软删除的帖子，按 created_at 倒序，最多返回 limit 条
	ListPublicPostsByAuthorIDs(ctx context.Context, authorIDs []uint64, limit int) ([]*model.Post, error)
}

// PostCommentRepository 帖子评论仓储接口
type PostCommentRepository interface {
	// CreateComment 创建评论
	CreateComment(ctx context.Context, comment *model.PostComment) error
	// GetCommentByID 根据 ID 查询评论（未软删除）
	GetCommentByID(ctx context.Context, id uint64) (*model.PostComment, error)
	// DeleteComment 软删除评论
	DeleteComment(ctx context.Context, id uint64) error
	// ListCommentsByPostID 分页查询帖子评论（一级评论，reply_to_id=0，支持排序）
	ListCommentsByPostID(ctx context.Context, postID uint64, offset, limit int, sortBy string) ([]*model.PostComment, int64, error)
	// ListChildCommentsByParentIDs 批量查询子评论（reply_to_id IN parentIDs）
	ListChildCommentsByParentIDs(ctx context.Context, postID uint64, parentIDs []uint64) ([]*model.PostComment, error)
	// UpdateComment 更新评论内容
	UpdateComment(ctx context.Context, id uint64, content string) error
	// IncrCommentLikeCount 评论点赞数原子增减（delta 为 +1 或 -1）
	IncrCommentLikeCount(ctx context.Context, id uint64, delta int) error
}

// PostLikeRepository 帖子点赞仓储接口
type PostLikeRepository interface {
	// HasPostLiked 判断用户是否已点赞帖子
	HasPostLiked(ctx context.Context, postID, userID uint64) (bool, error)
	// CreatePostLike 创建点赞记录
	CreatePostLike(ctx context.Context, like *model.PostLike) error
	// DeletePostLike 删除点赞记录
	DeletePostLike(ctx context.Context, postID, userID uint64) error
	// GetLikedPostIDs 批量查询用户对哪些帖子已点赞，返回点赞的帖子 ID 列表
	GetLikedPostIDs(ctx context.Context, userID uint64, postIDs []uint64) ([]uint64, error)
}

// PostCollectRepository 帖子收藏仓储接口
type PostCollectRepository interface {
	// HasPostCollected 判断用户是否已收藏帖子
	HasPostCollected(ctx context.Context, postID, userID uint64) (bool, error)
	// CreatePostCollect 创建收藏记录
	CreatePostCollect(ctx context.Context, collect *model.PostCollect) error
	// DeletePostCollect 删除收藏记录
	DeletePostCollect(ctx context.Context, postID, userID uint64) error
	// GetCollectedPostIDs 批量查询用户对哪些帖子已收藏，返回收藏的帖子 ID 列表
	GetCollectedPostIDs(ctx context.Context, userID uint64, postIDs []uint64) ([]uint64, error)
	// ListCollectedPostsByUserID 分页查询用户收藏的帖子 ID 列表
	ListCollectedPostsByUserID(ctx context.Context, userID uint64, offset, limit int) ([]uint64, int64, error)
}

// PostCommentLikeRepository 帖子评论点赞仓储接口
type PostCommentLikeRepository interface {
	// HasPostCommentLiked 判断用户是否已对评论点赞
	HasPostCommentLiked(ctx context.Context, commentID, userID uint64) (bool, error)
	// CreatePostCommentLike 创建评论点赞记录
	CreatePostCommentLike(ctx context.Context, like *model.PostCommentLike) error
	// DeletePostCommentLike 删除评论点赞记录
	DeletePostCommentLike(ctx context.Context, commentID, userID uint64) error
	// GetLikedPostCommentIDs 批量判断用户对哪些评论已点赞，返回 map[commentID]true
	GetLikedPostCommentIDs(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error)
}

// ReportRepository 内容举报仓储接口
type ReportRepository interface {
	// CreateReport 创建举报记录
	CreateReport(ctx context.Context, report *model.Report) error
	// HasReported 判断用户是否已举报过指定内容（防止重复举报）
	HasReported(ctx context.Context, reporterID uint64, targetType string, targetID uint64) (bool, error)
	// ListReportsByReporterID 分页查询用户的举报记录
	ListReportsByReporterID(ctx context.Context, reporterID uint64, offset, limit int) ([]*model.Report, int64, error)
	// EscalatePendingReports 将超过指定时间未处理的 pending 举报升级为 escalated
	EscalatePendingReports(ctx context.Context, olderThan time.Duration) (int64, error)
}

// ===========================
// 实现结构体
// ===========================

type topicRepository struct {
	db *gorm.DB
}

type postRepository struct {
	db *gorm.DB
}

type postCommentRepository struct {
	db *gorm.DB
}

type postLikeRepository struct {
	db *gorm.DB
}

type postCollectRepository struct {
	db *gorm.DB
}

type postCommentLikeRepository struct {
	db *gorm.DB
}

type reportRepository struct {
	db *gorm.DB
}

// ===========================
// 构造函数
// ===========================

// NewTopicRepository 创建话题仓储实例
func NewTopicRepository(db *gorm.DB) TopicRepository {
	return &topicRepository{db: db}
}

// NewPostRepository 创建帖子仓储实例
func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

// NewPostCommentRepository 创建帖子评论仓储实例
func NewPostCommentRepository(db *gorm.DB) PostCommentRepository {
	return &postCommentRepository{db: db}
}

// NewPostLikeRepository 创建帖子点赞仓储实例
func NewPostLikeRepository(db *gorm.DB) PostLikeRepository {
	return &postLikeRepository{db: db}
}

// NewPostCollectRepository 创建帖子收藏仓储实例
func NewPostCollectRepository(db *gorm.DB) PostCollectRepository {
	return &postCollectRepository{db: db}
}

// NewPostCommentLikeRepository 创建帖子评论点赞仓储实例
func NewPostCommentLikeRepository(db *gorm.DB) PostCommentLikeRepository {
	return &postCommentLikeRepository{db: db}
}

// NewReportRepository 创建举报仓储实例
func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

// ===========================
// TopicRepository 实现
// ===========================

// ListTopics 获取全量话题列表（按创建时间升序），post_count 为实时统计
func (r *topicRepository) ListTopics(ctx context.Context) ([]*model.Topic, error) {
	var topics []*model.Topic
	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&topics).Error; err != nil {
		return nil, fmt.Errorf("查询话题列表失败: %w", err)
	}
	// 用真实帖子数覆盖冗余 post_count，防止数据漂移
	postCounts, err := r.countPostsByTopic(ctx)
	if err == nil {
		for _, t := range topics {
			t.PostCount = postCounts[t.ID]
		}
	}
	return topics, nil
}

// countPostsByTopic 实时统计每个话题下的帖子数（未删除）
func (r *topicRepository) countPostsByTopic(ctx context.Context) (map[uint64]int, error) {
	type row struct {
		TopicID uint64 `gorm:"column:topic_id"`
		Count   int    `gorm:"column:cnt"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(
		`SELECT t.id AS topic_id, COUNT(p.id)::int AS cnt
		 FROM topics t
		 LEFT JOIN posts p ON p.deleted_at IS NULL AND p.topic_ids::jsonb @> to_jsonb(t.id)
		 GROUP BY t.id`,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint64]int, len(rows))
	for _, r := range rows {
		result[r.TopicID] = r.Count
	}
	return result, nil
}

// GetTopicByID 根据 ID 查询话题
func (r *topicRepository) GetTopicByID(ctx context.Context, id uint64) (*model.Topic, error) {
	var topic model.Topic
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&topic).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.Newf(apperrors.CodeNotFound, "话题 ID %d 不存在", id)
		}
		return nil, fmt.Errorf("查询话题失败: %w", err)
	}
	return &topic, nil
}

// GetTopicsByIDs 按 ID 列表批量查询话题
func (r *topicRepository) GetTopicsByIDs(ctx context.Context, ids []uint64) ([]*model.Topic, error) {
	if len(ids) == 0 {
		return []*model.Topic{}, nil
	}
	var topics []*model.Topic
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&topics).Error; err != nil {
		return nil, fmt.Errorf("批量查询话题失败: %w", err)
	}
	return topics, nil
}

// IncrPostCount 原子增减话题的帖子计数
func (r *topicRepository) IncrPostCount(ctx context.Context, id uint64, delta int) error {
	if delta > 0 {
		return r.db.WithContext(ctx).
			Model(&model.Topic{}).
			Where("id = ?", id).
			Update("post_count", gorm.Expr("post_count + ?", delta)).Error
	}
	// 减少时保证不低于 0
	return r.db.WithContext(ctx).
		Model(&model.Topic{}).
		Where("id = ? AND post_count > 0", id).
		Update("post_count", gorm.Expr("post_count + ?", delta)).Error
}

// ===========================
// PostRepository 实现
// ===========================

// CreatePost 创建帖子
func (r *postRepository) CreatePost(ctx context.Context, post *model.Post) error {
	if err := r.db.WithContext(ctx).Create(post).Error; err != nil {
		return fmt.Errorf("创建帖子失败: %w", err)
	}
	return nil
}

// GetPostByID 根据 ID 查询帖子（未软删除）
func (r *postRepository) GetPostByID(ctx context.Context, id uint64) (*model.Post, error) {
	var post model.Post
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodePostNotFound)
		}
		return nil, fmt.Errorf("查询帖子失败: %w", err)
	}
	return &post, nil
}

// GetPostsByIDs 按 ID 列表批量获取帖子（跳过软删除条目）
func (r *postRepository) GetPostsByIDs(ctx context.Context, ids []uint64) ([]*model.Post, error) {
	if len(ids) == 0 {
		return []*model.Post{}, nil
	}
	// 按传入的 ID 顺序返回
	var posts []*model.Post
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("批量查询帖子失败: %w", err)
	}
	// 按原始 ID 顺序排序
	postMap := make(map[uint64]*model.Post, len(posts))
	for _, p := range posts {
		postMap[p.ID] = p
	}
	ordered := make([]*model.Post, 0, len(ids))
	for _, id := range ids {
		if p, ok := postMap[id]; ok {
			ordered = append(ordered, p)
		}
	}
	return ordered, nil
}

// UpdatePost 更新帖子指定字段
func (r *postRepository) UpdatePost(ctx context.Context, id uint64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新帖子失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodePostNotFound)
	}
	return nil
}

// DeletePost 软删除帖子
func (r *postRepository) DeletePost(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("NOW()"))
	if result.Error != nil {
		return fmt.Errorf("删除帖子失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodePostNotFound)
	}
	return nil
}

// ListPosts 分页查询帖子列表（支持话题/项目/作者筛选、关键词搜索、排序）
func (r *postRepository) ListPosts(ctx context.Context, params *ListPostsParams) ([]*model.Post, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("deleted_at IS NULL")

	// 按话题 ID 筛选（JSONB 查询，使用 @> 操作符避免 GORM ? 占位符冲突）
	if params.TopicID > 0 {
		query = query.Where("topic_ids::jsonb @> ?::jsonb", fmt.Sprintf("[%d]", params.TopicID))
	}

	// 按项目 ID 筛选
	if params.ProjectID > 0 {
		query = query.Where("project_id = ?", params.ProjectID)
	}

	// 按作者 ID 筛选
	if params.AuthorID > 0 {
		query = query.Where("author_id = ?", params.AuthorID)
	}

	// 关键词搜索（同时匹配 title 和 content）
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("(title ILIKE ? OR content ILIKE ?)", keyword, keyword)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计帖子总数失败: %w", err)
	}

	// 排序
	orderClause := "is_pinned DESC, created_at DESC"
	if params.SortBy == "hot" {
		orderClause = "is_pinned DESC, hot_score DESC, created_at DESC"
	}

	// 分页查询
	var posts []*model.Post
	if err := query.
		Order(orderClause).
		Offset(params.Offset).
		Limit(params.Limit).
		Find(&posts).Error; err != nil {
		return nil, 0, fmt.Errorf("查询帖子列表失败: %w", err)
	}
	return posts, total, nil
}

// ListHotPosts 查询热门帖子（按热度分数倒序，置顶优先）
func (r *postRepository) ListHotPosts(ctx context.Context, limit int) ([]*model.Post, error) {
	var posts []*model.Post
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("is_pinned DESC, hot_score DESC, created_at DESC").
		Limit(limit).
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询热门帖子失败: %w", err)
	}
	return posts, nil
}

// UpdateCounts 原子增减帖子冗余数字段
func (r *postRepository) UpdateCounts(ctx context.Context, id uint64, field string, delta int) error {
	if delta > 0 {
		return r.db.WithContext(ctx).
			Model(&model.Post{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Update(field, gorm.Expr("? + ?", gorm.Expr(field), delta)).Error
	}
	// 减少时保证不低于 0
	return r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("id = ? AND deleted_at IS NULL AND "+field+" > 0", id).
		Update(field, gorm.Expr("? + ?", gorm.Expr(field), delta)).Error
}

// IncrViewCount 浏览数 +1（原子操作）
func (r *postRepository) IncrViewCount(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

// ListPostsForHotUpdate 查询需要刷新热度分数的帖子列表（最近 30 天内创建或有互动）
func (r *postRepository) ListPostsForHotUpdate(ctx context.Context, limit int) ([]*model.Post, error) {
	var posts []*model.Post
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND created_at > NOW() - INTERVAL '30 days'").
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询热度更新帖子列表失败: %w", err)
	}
	return posts, nil
}

// BatchUpdateHotScores 批量更新帖子热度分数
func (r *postRepository) BatchUpdateHotScores(ctx context.Context, updates map[uint64]float64) error {
	if len(updates) == 0 {
		return nil
	}
	for id, score := range updates {
		if err := r.db.WithContext(ctx).
			Model(&model.Post{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Update("hot_score", score).Error; err != nil {
			return fmt.Errorf("批量更新热度分数失败: %w", err)
		}
	}
	return nil
}

// ListActivePostsWithinDays 查询最近 N 天内有互动（点赞/评论）的帖子
func (r *postRepository) ListActivePostsWithinDays(ctx context.Context, days int, limit int) ([]*model.Post, error) {
	var posts []*model.Post
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND (like_count > 0 OR comment_count > 0) AND created_at > NOW() - (? * INTERVAL '1 day')", days).
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询活跃帖子失败: %w", err)
	}
	return posts, nil
}

// ListPublicPostsByAuthorIDs 批量查询多个作者的公开帖子（用于动态流）
// 查询 author_id IN authorIDs 且未软删除的帖子，按 created_at 倒序，最多返回 limit 条
func (r *postRepository) ListPublicPostsByAuthorIDs(ctx context.Context, authorIDs []uint64, limit int) ([]*model.Post, error) {
	if len(authorIDs) == 0 {
		return []*model.Post{}, nil
	}
	var posts []*model.Post
	if err := r.db.WithContext(ctx).
		Where("author_id IN ? AND deleted_at IS NULL", authorIDs).
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("查询动态流帖子失败: %w", err)
	}
	return posts, nil
}

// ===========================
// PostCommentRepository 实现
// ===========================

// CreateComment 创建评论
func (r *postCommentRepository) CreateComment(ctx context.Context, comment *model.PostComment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return fmt.Errorf("创建帖子评论失败: %w", err)
	}
	return nil
}

// GetCommentByID 根据 ID 查询评论（未软删除）
func (r *postCommentRepository) GetCommentByID(ctx context.Context, id uint64) (*model.PostComment, error) {
	var comment model.PostComment
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeCommentNotFound)
		}
		return nil, fmt.Errorf("查询帖子评论失败: %w", err)
	}
	return &comment, nil
}

// DeleteComment 软删除评论
func (r *postCommentRepository) DeleteComment(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.PostComment{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("NOW()"))
	if result.Error != nil {
		return fmt.Errorf("删除帖子评论失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeCommentNotFound)
	}
	return nil
}

// ListCommentsByPostID 分页查询帖子的一级评论（reply_to_id = 0），支持排序
func (r *postCommentRepository) ListCommentsByPostID(ctx context.Context, postID uint64, offset, limit int, sortBy string) ([]*model.PostComment, int64, error) {
	// 统计一级评论总数
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.PostComment{}).
		Where("post_id = ? AND reply_to_id = 0 AND deleted_at IS NULL", postID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计评论总数失败: %w", err)
	}

	// 排序方式
	orderClause := "created_at ASC"
	if sortBy == "hot" {
		orderClause = "like_count DESC, created_at DESC"
	}

	// 查询一级评论
	var comments []*model.PostComment
	if err := r.db.WithContext(ctx).
		Where("post_id = ? AND reply_to_id = 0 AND deleted_at IS NULL", postID).
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询评论列表失败: %w", err)
	}
	return comments, total, nil
}

// ListChildCommentsByParentIDs 批量查询子评论（reply_to_id IN parentIDs）
func (r *postCommentRepository) ListChildCommentsByParentIDs(ctx context.Context, postID uint64, parentIDs []uint64) ([]*model.PostComment, error) {
	if len(parentIDs) == 0 {
		return []*model.PostComment{}, nil
	}
	var comments []*model.PostComment
	if err := r.db.WithContext(ctx).
		Where("post_id = ? AND reply_to_id IN ? AND deleted_at IS NULL", postID, parentIDs).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("查询子评论失败: %w", err)
	}
	return comments, nil
}

// UpdateComment 更新评论内容
func (r *postCommentRepository) UpdateComment(ctx context.Context, id uint64, content string) error {
	result := r.db.WithContext(ctx).
		Model(&model.PostComment{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("content", content)
	if result.Error != nil {
		return fmt.Errorf("更新帖子评论失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeCommentNotFound)
	}
	return nil
}

// IncrCommentLikeCount 评论点赞数原子增减（delta 为 +1 或 -1，最小不小于 0）
func (r *postCommentRepository) IncrCommentLikeCount(ctx context.Context, id uint64, delta int) error {
	if delta > 0 {
		return r.db.WithContext(ctx).
			Model(&model.PostComment{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Update("like_count", gorm.Expr("like_count + ?", delta)).Error
	}
	// 减少时保证不低于 0
	return r.db.WithContext(ctx).
		Model(&model.PostComment{}).
		Where("id = ? AND deleted_at IS NULL AND like_count > 0", id).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// ===========================
// PostLikeRepository 实现
// ===========================

// HasPostLiked 判断用户是否已点赞帖子
func (r *postLikeRepository) HasPostLiked(ctx context.Context, postID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	return count > 0, err
}

// CreatePostLike 创建点赞记录
func (r *postLikeRepository) CreatePostLike(ctx context.Context, like *model.PostLike) error {
	if err := r.db.WithContext(ctx).Create(like).Error; err != nil {
		return fmt.Errorf("创建帖子点赞记录失败: %w", err)
	}
	return nil
}

// DeletePostLike 删除点赞记录
func (r *postLikeRepository) DeletePostLike(ctx context.Context, postID, userID uint64) error {
	result := r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.PostLike{})
	if result.Error != nil {
		return fmt.Errorf("删除帖子点赞记录失败: %w", result.Error)
	}
	return nil
}

// GetLikedPostIDs 批量查询用户对哪些帖子已点赞，返回点赞的帖子 ID 列表
func (r *postLikeRepository) GetLikedPostIDs(ctx context.Context, userID uint64, postIDs []uint64) ([]uint64, error) {
	if len(postIDs) == 0 {
		return []uint64{}, nil
	}
	var likedIDs []uint64
	err := r.db.WithContext(ctx).
		Model(&model.PostLike{}).
		Select("post_id").
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Pluck("post_id", &likedIDs).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询帖子点赞状态失败: %w", err)
	}
	return likedIDs, nil
}

// ===========================
// PostCollectRepository 实现
// ===========================

// HasPostCollected 判断用户是否已收藏帖子
func (r *postCollectRepository) HasPostCollected(ctx context.Context, postID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PostCollect{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	return count > 0, err
}

// CreatePostCollect 创建收藏记录
func (r *postCollectRepository) CreatePostCollect(ctx context.Context, collect *model.PostCollect) error {
	if err := r.db.WithContext(ctx).Create(collect).Error; err != nil {
		return fmt.Errorf("创建帖子收藏记录失败: %w", err)
	}
	return nil
}

// DeletePostCollect 删除收藏记录
func (r *postCollectRepository) DeletePostCollect(ctx context.Context, postID, userID uint64) error {
	result := r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.PostCollect{})
	if result.Error != nil {
		return fmt.Errorf("删除帖子收藏记录失败: %w", result.Error)
	}
	return nil
}

// GetCollectedPostIDs 批量查询用户对哪些帖子已收藏，返回收藏的帖子 ID 列表
func (r *postCollectRepository) GetCollectedPostIDs(ctx context.Context, userID uint64, postIDs []uint64) ([]uint64, error) {
	if len(postIDs) == 0 {
		return []uint64{}, nil
	}
	var collectedIDs []uint64
	err := r.db.WithContext(ctx).
		Model(&model.PostCollect{}).
		Select("post_id").
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Pluck("post_id", &collectedIDs).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询帖子收藏状态失败: %w", err)
	}
	return collectedIDs, nil
}

// ListCollectedPostsByUserID 分页查询用户收藏的帖子 ID 列表
func (r *postCollectRepository) ListCollectedPostsByUserID(ctx context.Context, userID uint64, offset, limit int) ([]uint64, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.PostCollect{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计帖子收藏数失败: %w", err)
	}

	var collects []model.PostCollect
	if err := r.db.WithContext(ctx).Model(&model.PostCollect{}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&collects).Error; err != nil {
		return nil, 0, fmt.Errorf("查询帖子收藏列表失败: %w", err)
	}

	postIDs := make([]uint64, 0, len(collects))
	for _, c := range collects {
		postIDs = append(postIDs, c.PostID)
	}
	return postIDs, total, nil
}

// ===========================
// PostCommentLikeRepository 实现
// ===========================

// HasPostCommentLiked 判断用户是否已对评论点赞
func (r *postCommentLikeRepository) HasPostCommentLiked(ctx context.Context, commentID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PostCommentLike{}).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Count(&count).Error
	return count > 0, err
}

// CreatePostCommentLike 创建评论点赞记录
func (r *postCommentLikeRepository) CreatePostCommentLike(ctx context.Context, like *model.PostCommentLike) error {
	if err := r.db.WithContext(ctx).Create(like).Error; err != nil {
		return fmt.Errorf("创建帖子评论点赞记录失败: %w", err)
	}
	return nil
}

// DeletePostCommentLike 删除评论点赞记录
func (r *postCommentLikeRepository) DeletePostCommentLike(ctx context.Context, commentID, userID uint64) error {
	result := r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Delete(&model.PostCommentLike{})
	if result.Error != nil {
		return fmt.Errorf("删除帖子评论点赞记录失败: %w", result.Error)
	}
	return nil
}

// GetLikedPostCommentIDs 批量判断用户对哪些评论已点赞，返回 map[commentID]true
func (r *postCommentLikeRepository) GetLikedPostCommentIDs(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	if len(commentIDs) == 0 {
		return map[uint64]bool{}, nil
	}
	var likedIDs []uint64
	err := r.db.WithContext(ctx).
		Model(&model.PostCommentLike{}).
		Select("comment_id").
		Where("user_id = ? AND comment_id IN ?", userID, commentIDs).
		Pluck("comment_id", &likedIDs).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询帖子评论点赞状态失败: %w", err)
	}
	result := make(map[uint64]bool, len(likedIDs))
	for _, id := range likedIDs {
		result[id] = true
	}
	return result, nil
}

// ===========================
// ReportRepository 实现
// ===========================

// CreateReport 创建举报记录
func (r *reportRepository) CreateReport(ctx context.Context, report *model.Report) error {
	if err := r.db.WithContext(ctx).Create(report).Error; err != nil {
		return fmt.Errorf("创建举报记录失败: %w", err)
	}
	return nil
}

// HasReported 判断用户是否已举报过指定内容
func (r *reportRepository) HasReported(ctx context.Context, reporterID uint64, targetType string, targetID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Report{}).
		Where("reporter_id = ? AND target_type = ? AND target_id = ?", reporterID, targetType, targetID).
		Count(&count).Error
	return count > 0, err
}

// ListReportsByReporterID 分页查询用户的举报记录
func (r *reportRepository) ListReportsByReporterID(ctx context.Context, reporterID uint64, offset, limit int) ([]*model.Report, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Report{}).
		Where("reporter_id = ?", reporterID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计举报记录数失败: %w", err)
	}

	var reports []*model.Report
	if err := r.db.WithContext(ctx).
		Where("reporter_id = ?", reporterID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&reports).Error; err != nil {
		return nil, 0, fmt.Errorf("查询举报记录列表失败: %w", err)
	}
	return reports, total, nil
}

// EscalatePendingReports 将超过指定时间未处理的 pending 举报升级为 escalated
func (r *reportRepository) EscalatePendingReports(ctx context.Context, olderThan time.Duration) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&model.Report{}).
		Where("status = ? AND created_at < ?", "pending", time.Now().Add(-olderThan)).
		Update("status", "escalated")
	if result.Error != nil {
		return 0, fmt.Errorf("升级举报状态失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}
