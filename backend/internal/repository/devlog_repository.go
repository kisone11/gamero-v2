// Package repository 提供开发日志系统的数据库访问层。
// 采用接口 + 实现的模式，封装所有对 DevLog、DevLogComment、DevLogLike、
// DevLogCollect、DevLogCommentLike 表的操作。
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gamero/gamero/internal/model"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"gorm.io/gorm"
)

// ===========================
// 接口定义
// ===========================

// DevLogRepository 开发日志仓储接口
type DevLogRepository interface {
	// CreateLog 创建开发日志，返回含 ID 的日志对象
	CreateLog(ctx context.Context, log *model.DevLog) error
	// GetLogByID 根据 ID 查询日志（未软删除）
	GetLogByID(ctx context.Context, id uint64) (*model.DevLog, error)
	// GetLogsByIDs 按 ID 列表批量获取日志（Feed 渲染用，跳过软删除条目）
	GetLogsByIDs(ctx context.Context, ids []uint64) ([]*model.DevLog, error)
	// UpdateLog 更新日志指定字段
	UpdateLog(ctx context.Context, id uint64, updates map[string]interface{}) error
	// DeleteLog 软删除日志
	DeleteLog(ctx context.Context, id uint64) error
	// ListLogsByProjectID 分页查询项目下的日志列表，按发布时间倒序
	ListLogsByProjectID(ctx context.Context, projectID uint64, params *ListDevLogsParams) ([]*model.DevLog, int64, error)
	// ListReleasesByProjectID 查询项目下所有版本更新日志，按版本号倒序
	ListReleasesByProjectID(ctx context.Context, projectID uint64, offset, limit int) ([]*model.DevLog, int64, error)
	// CountReleasesByProjectID 统计项目的版本发布数量（log_type='release' 且已发布且未软删除）
	CountReleasesByProjectID(ctx context.Context, projectID uint64) (int64, error)
	// SumDownloadsByProjectID 统计项目所有 release 日志的下载总数
	SumDownloadsByProjectID(ctx context.Context, projectID uint64) (int64, error)
	// ListPublicLogsByAuthorIDs 批量查询多个作者的公开已发布日志（用于动态流）
	// 按 created_at 倒序，最多返回 limit 条
	ListPublicLogsByAuthorIDs(ctx context.Context, authorIDs []uint64, limit int) ([]*model.DevLog, error)
	// ListPublicLogsByProjectIDs batch query public published logs by project IDs (all types, for Feed)
	ListPublicLogsByProjectIDs(ctx context.Context, projectIDs []uint64, limit int) ([]*model.DevLog, error)
	// ListPublicLogsByProjectIDsAndType batch query public published logs by project IDs and log type (for Feed)
	// Ordered by created_at DESC, max limit items.
	ListPublicLogsByProjectIDsAndType(ctx context.Context, projectIDs []uint64, logType model.DevLogType, limit int) ([]*model.DevLog, error)
	// ListLogsByAuthorID 分页查询指定作者的已发布日志列表（个人主页用）
	ListLogsByAuthorID(ctx context.Context, authorID uint64, offset, limit int) ([]*model.DevLog, int64, error)
	// IncrViewCount 浏览数 +1（原子操作）
	IncrViewCount(ctx context.Context, id uint64) error
	// UpdateCounts 更新冗余计数字段（like_count / comment_count / collect_count）
	UpdateCounts(ctx context.Context, id uint64, updates map[string]interface{}) error

	// ===== 高级查询 =====

	// GetProjectLogStats 按项目聚合日志统计
	GetProjectLogStats(ctx context.Context, projectID uint64) (*ProjectLogStats, error)
	// GetLogTimeline 日志时间线(按月聚合数量)
	GetLogTimeline(ctx context.Context, projectID uint64) ([]MonthlyLogCount, error)
	// ListHotLogs 热门日志排行
	ListHotLogs(ctx context.Context, days int, logType string, page, pageSize int) ([]*model.DevLog, int64, error)
	// GetRelatedLogs 相关日志推荐(同项目/同作者/相似标签)
	GetRelatedLogs(ctx context.Context, logID uint64, limit int) ([]*model.DevLog, error)
}

// DevLogCommentRepository 日志评论仓储接口
type DevLogCommentRepository interface {
	// CreateComment 创建评论
	CreateComment(ctx context.Context, comment *model.DevLogComment) error
	// GetCommentByID 根据 ID 查询评论（未软删除）
	GetCommentByID(ctx context.Context, id uint64) (*model.DevLogComment, error)
	// DeleteComment 软删除评论
	DeleteComment(ctx context.Context, id uint64) error
	// UpdateComment 更新评论内容
	UpdateComment(ctx context.Context, id uint64, content string) error
	// ListCommentsByLogID 分页查询日志下的评论（一级评论，支持排序，子评论挂载在父评论下）
	// 返回所有一级评论及其子评论（不分页子评论，业务上子评论数量有限）
	ListCommentsByLogID(ctx context.Context, logID uint64, offset, limit int, sortBy string) ([]*model.DevLogComment, int64, error)
	// ListChildCommentsByParentIDs 批量查询父评论的子评论（reply_to_id in parentIDs）
	ListChildCommentsByParentIDs(ctx context.Context, logID uint64, parentIDs []uint64) ([]*model.DevLogComment, error)
	// CountCommentsByLogID 统计日志下的评论总数（含子评论）
	CountCommentsByLogID(ctx context.Context, logID uint64) (int64, error)
	// IncrCommentLikeCount 评论点赞数原子增减（delta 为 +1 或 -1）
	IncrCommentLikeCount(ctx context.Context, commentID uint64, delta int) error

	// ===== 高级查询 =====

	// GetLogCommentTree 评论树(楼中楼)
	GetLogCommentTree(ctx context.Context, logID uint64, page, pageSize int) ([]*LogCommentNode, int64, error)
	// SearchLogComments 日志评论全文搜索
	SearchLogComments(ctx context.Context, logID uint64, query string) ([]*model.DevLogComment, error)
}

// DevLogLikeRepository 日志点赞仓储接口
type DevLogLikeRepository interface {
	// CreateLike 创建点赞记录
	CreateLike(ctx context.Context, like *model.DevLogLike) error
	// DeleteLike 删除点赞记录
	DeleteLike(ctx context.Context, logID, userID uint64) error
	// HasLiked 判断用户是否已点赞
	HasLiked(ctx context.Context, logID, userID uint64) (bool, error)
	// CountLikes 统计日志点赞总数
	CountLikes(ctx context.Context, logID uint64) (int64, error)
	// GetLikedLogIDs 批量查询用户对指定日志列表中已点赞的 ID 集合
	GetLikedLogIDs(ctx context.Context, userID uint64, logIDs []uint64) (map[uint64]bool, error)
}

// DevLogCollectRepository 日志收藏仓储接口
type DevLogCollectRepository interface {
	// CreateCollect 创建收藏记录
	CreateCollect(ctx context.Context, collect *model.DevLogCollect) error
	// DeleteCollect 删除收藏记录
	DeleteCollect(ctx context.Context, logID, userID uint64) error
	// HasCollected 判断用户是否已收藏
	HasCollected(ctx context.Context, logID, userID uint64) (bool, error)
	// ListCollectedLogsByUserID 分页查询用户收藏的日志 ID 列表
	ListCollectedLogsByUserID(ctx context.Context, userID uint64, offset, limit int) ([]uint64, int64, error)
	// GetCollectedLogIDs 批量查询用户对指定日志列表中已收藏的 ID 集合
	GetCollectedLogIDs(ctx context.Context, userID uint64, logIDs []uint64) (map[uint64]bool, error)
}

// DevLogCommentLikeRepository 评论点赞仓储接口
type DevLogCommentLikeRepository interface {
	// CreateCommentLike 创建评论点赞记录
	CreateCommentLike(ctx context.Context, like *model.DevLogCommentLike) error
	// DeleteCommentLike 删除评论点赞记录
	DeleteCommentLike(ctx context.Context, commentID, userID uint64) error
	// HasCommentLiked 判断用户是否已对评论点赞
	HasCommentLiked(ctx context.Context, commentID, userID uint64) (bool, error)
	// GetLikedCommentIDs 批量判断用户对哪些评论已点赞，返回 map[commentID]true
	GetLikedCommentIDs(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error)
}

// ListDevLogsParams 日志列表查询参数
type ListDevLogsParams struct {
	LogType    model.DevLogType       // 按类型筛选（空字符串表示不筛选）
	Status     model.DevLogStatus     // 按状态筛选（空字符串表示不筛选）
	Visibility model.DevLogVisibility // 按可见范围筛选（空字符串表示不筛选）
	Offset     int
	Limit      int
}

// ===========================
// 实现结构体
// ===========================

// devLogRepository 开发日志仓储实现
type devLogRepository struct {
	db *gorm.DB
}

// devLogCommentRepository 日志评论仓储实现
type devLogCommentRepository struct {
	db *gorm.DB
}

// devLogLikeRepository 日志点赞仓储实现
type devLogLikeRepository struct {
	db *gorm.DB
}

// devLogCollectRepository 日志收藏仓储实现
type devLogCollectRepository struct {
	db *gorm.DB
}

// devLogCommentLikeRepository 评论点赞仓储实现
type devLogCommentLikeRepository struct {
	db *gorm.DB
}

// ===========================
// 构造函数
// ===========================

// NewDevLogRepository 创建开发日志仓储实例
func NewDevLogRepository(db *gorm.DB) DevLogRepository {
	return &devLogRepository{db: db}
}

// NewDevLogCommentRepository 创建日志评论仓储实例
func NewDevLogCommentRepository(db *gorm.DB) DevLogCommentRepository {
	return &devLogCommentRepository{db: db}
}

// NewDevLogLikeRepository 创建日志点赞仓储实例
func NewDevLogLikeRepository(db *gorm.DB) DevLogLikeRepository {
	return &devLogLikeRepository{db: db}
}

// NewDevLogCollectRepository 创建日志收藏仓储实例
func NewDevLogCollectRepository(db *gorm.DB) DevLogCollectRepository {
	return &devLogCollectRepository{db: db}
}

// NewDevLogCommentLikeRepository 创建评论点赞仓储实例
func NewDevLogCommentLikeRepository(db *gorm.DB) DevLogCommentLikeRepository {
	return &devLogCommentLikeRepository{db: db}
}

// ===========================
// DevLogRepository 实现
// ===========================

// CreateLog 创建开发日志
func (r *devLogRepository) CreateLog(ctx context.Context, log *model.DevLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("创建开发日志失败: %w", err)
	}
	return nil
}

// GetLogByID 根据 ID 查询日志
func (r *devLogRepository) GetLogByID(ctx context.Context, id uint64) (*model.DevLog, error) {
	var log model.DevLog
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeDevLogNotFound)
		}
		return nil, fmt.Errorf("查询开发日志失败: %w", err)
	}
	return &log, nil
}

// GetLogsByIDs 按 ID 列表批量获取日志（Feed 渲染用，跳过已软删除条目）
func (r *devLogRepository) GetLogsByIDs(ctx context.Context, ids []uint64) ([]*model.DevLog, error) {
	if len(ids) == 0 {
		return []*model.DevLog{}, nil
	}
	var logs []*model.DevLog
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("批量查询开发日志失败: %w", err)
	}
	return logs, nil
}


func (r *devLogRepository) UpdateLog(ctx context.Context, id uint64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.DevLog{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新开发日志失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeDevLogNotFound)
	}
	return nil
}

// DeleteLog 软删除日志
func (r *devLogRepository) DeleteLog(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.DevLog{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("NOW()"))
	if result.Error != nil {
		return fmt.Errorf("删除开发日志失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeDevLogNotFound)
	}
	return nil
}

// ListLogsByProjectID 分页查询项目下的日志列表
func (r *devLogRepository) ListLogsByProjectID(ctx context.Context, projectID uint64, params *ListDevLogsParams) ([]*model.DevLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.DevLog{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID)

	// 按类型筛选
	if params.LogType != "" {
		query = query.Where("log_type = ?", params.LogType)
	}
	// 按状态筛选
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	// 按可见范围筛选
	if params.Visibility != "" {
		query = query.Where("visibility = ?", params.Visibility)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计日志总数失败: %w", err)
	}

	// 按发布时间倒序
	var logs []*model.DevLog
	if err := query.
		Order("created_at DESC").
		Offset(params.Offset).
		Limit(params.Limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询日志列表失败: %w", err)
	}
	return logs, total, nil
}

// CountReleasesByProjectID 统计项目的版本发布数量（log_type='release' 且已发布且未软删除）
func (r *devLogRepository) CountReleasesByProjectID(ctx context.Context, projectID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DevLog{}).
		Where("project_id = ? AND log_type = ? AND status = ? AND deleted_at IS NULL",
			projectID, model.DevLogTypeRelease, model.DevLogStatusPublished).
		Count(&count).Error
	return count, err
}

// SumDownloadsByProjectID 统计项目所有 release 日志的下载总数
func (r *devLogRepository) SumDownloadsByProjectID(ctx context.Context, projectID uint64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.DevLog{}).
		Where("project_id = ? AND log_type = ? AND deleted_at IS NULL",
			projectID, model.DevLogTypeRelease).
		Select("COALESCE(SUM(download_count), 0)").
		Scan(&total).Error
	return total, err
}

// ListReleasesByProjectID 查询版本更新日志，按版本号倒序
func (r *devLogRepository) ListReleasesByProjectID(ctx context.Context, projectID uint64, offset, limit int) ([]*model.DevLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.DevLog{}).
		Where("project_id = ? AND log_type = ? AND status = ? AND deleted_at IS NULL",
			projectID, model.DevLogTypeRelease, model.DevLogStatusPublished)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计版本日志总数失败: %w", err)
	}

	// 按版本号倒序（字典序倒序，对语义版本号 v0.x.x 效果较好）
	var logs []*model.DevLog
	if err := query.
		Order("version DESC, created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询版本日志列表失败: %w", err)
	}
	return logs, total, nil
}

// ListPublicLogsByAuthorIDs 批量查询多个作者的公开已发布日志（用于动态流）
// 查询 author_id IN authorIDs 且 status=published、visibility=public、未软删除的日志
// 按 created_at 倒序，最多返回 limit 条
func (r *devLogRepository) ListPublicLogsByAuthorIDs(ctx context.Context, authorIDs []uint64, limit int) ([]*model.DevLog, error) {
	if len(authorIDs) == 0 {
		return []*model.DevLog{}, nil
	}
	var logs []*model.DevLog
	if err := r.db.WithContext(ctx).
		Where("author_id IN ? AND status = ? AND visibility = ? AND deleted_at IS NULL",
			authorIDs, model.DevLogStatusPublished, model.DevLogVisibilityPublic).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("查询动态流日志失败: %w", err)
	}
	return logs, nil
}

// ListPublicLogsByProjectIDsAndType batch query public published logs by project IDs and log type (for Feed)
func (r *devLogRepository) ListPublicLogsByProjectIDsAndType(ctx context.Context, projectIDs []uint64, logType model.DevLogType, limit int) ([]*model.DevLog, error) {
	if len(projectIDs) == 0 {
		return []*model.DevLog{}, nil
	}
	var logs []*model.DevLog
	if err := r.db.WithContext(ctx).
		Where("project_id IN ? AND status = ? AND visibility = ? AND log_type = ? AND deleted_at IS NULL",
			projectIDs, model.DevLogStatusPublished, model.DevLogVisibilityPublic, logType).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("batch query project devlogs failed: %w", err)
	}
	return logs, nil
}

// ListPublicLogsByProjectIDs batch query all public published logs by project IDs (for Feed, all types)
func (r *devLogRepository) ListPublicLogsByProjectIDs(ctx context.Context, projectIDs []uint64, limit int) ([]*model.DevLog, error) {
	if len(projectIDs) == 0 {
		return []*model.DevLog{}, nil
	}
	var logs []*model.DevLog
	if err := r.db.WithContext(ctx).
		Where("project_id IN ? AND status = ? AND visibility = ? AND deleted_at IS NULL",
			projectIDs, model.DevLogStatusPublished, model.DevLogVisibilityPublic).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("batch query project devlogs failed: %w", err)
	}
	return logs, nil
}

// IncrViewCount 浏览数原子 +1
func (r *devLogRepository) IncrViewCount(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.DevLog{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

// UpdateCounts 更新冗余计数字段
func (r *devLogRepository) UpdateCounts(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&model.DevLog{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates).Error
}

// ===========================
// DevLogCommentRepository 实现
// ===========================

// CreateComment 创建评论
func (r *devLogCommentRepository) CreateComment(ctx context.Context, comment *model.DevLogComment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return fmt.Errorf("创建评论失败: %w", err)
	}
	return nil
}

// GetCommentByID 根据 ID 查询评论
func (r *devLogCommentRepository) GetCommentByID(ctx context.Context, id uint64) (*model.DevLogComment, error) {
	var comment model.DevLogComment
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeCommentNotFound)
		}
		return nil, fmt.Errorf("查询评论失败: %w", err)
	}
	return &comment, nil
}

// DeleteComment 软删除评论
func (r *devLogCommentRepository) DeleteComment(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.DevLogComment{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", gorm.Expr("NOW()"))
	if result.Error != nil {
		return fmt.Errorf("删除评论失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeCommentNotFound)
	}
	return nil
}

// UpdateComment 更新评论内容
func (r *devLogCommentRepository) UpdateComment(ctx context.Context, id uint64, content string) error {
	result := r.db.WithContext(ctx).
		Model(&model.DevLogComment{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("content", content)
	if result.Error != nil {
		return fmt.Errorf("更新评论失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeCommentNotFound)
	}
	return nil
}

// ListCommentsByLogID 分页查询日志的一级评论（reply_to_id = 0），支持排序
func (r *devLogCommentRepository) ListCommentsByLogID(ctx context.Context, logID uint64, offset, limit int, sortBy string) ([]*model.DevLogComment, int64, error) {
	// 仅统计一级评论数量
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.DevLogComment{}).
		Where("log_id = ? AND reply_to_id = 0 AND deleted_at IS NULL", logID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计评论总数失败: %w", err)
	}

	// 排序方式
	orderClause := "created_at ASC"
	if sortBy == "hot" {
		orderClause = "like_count DESC, created_at DESC"
	}

	// 查询一级评论
	var comments []*model.DevLogComment
	if err := r.db.WithContext(ctx).
		Where("log_id = ? AND reply_to_id = 0 AND deleted_at IS NULL", logID).
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询评论列表失败: %w", err)
	}
	return comments, total, nil
}

// ListChildCommentsByParentIDs 批量查询子评论
func (r *devLogCommentRepository) ListChildCommentsByParentIDs(ctx context.Context, logID uint64, parentIDs []uint64) ([]*model.DevLogComment, error) {
	if len(parentIDs) == 0 {
		return []*model.DevLogComment{}, nil
	}
	var comments []*model.DevLogComment
	if err := r.db.WithContext(ctx).
		Where("log_id = ? AND reply_to_id IN ? AND deleted_at IS NULL", logID, parentIDs).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("查询子评论失败: %w", err)
	}
	return comments, nil
}

// CountCommentsByLogID 统计日志下评论总数（含子评论）
func (r *devLogCommentRepository) CountCommentsByLogID(ctx context.Context, logID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DevLogComment{}).
		Where("log_id = ? AND deleted_at IS NULL", logID).
		Count(&count).Error
	return count, err
}

// IncrCommentLikeCount 评论点赞数原子增减（delta 为 +1 或 -1，最小不小于 0）
func (r *devLogCommentRepository) IncrCommentLikeCount(ctx context.Context, commentID uint64, delta int) error {
	if delta > 0 {
		return r.db.WithContext(ctx).
			Model(&model.DevLogComment{}).
			Where("id = ? AND deleted_at IS NULL", commentID).
			Update("like_count", gorm.Expr("like_count + ?", delta)).Error
	}
	// 减少时保证不低于 0
	return r.db.WithContext(ctx).
		Model(&model.DevLogComment{}).
		Where("id = ? AND deleted_at IS NULL AND like_count > 0", commentID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// ===========================
// DevLogLikeRepository 实现
// ===========================

// CreateLike 创建点赞记录
func (r *devLogLikeRepository) CreateLike(ctx context.Context, like *model.DevLogLike) error {
	if err := r.db.WithContext(ctx).Create(like).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") ||
			strings.Contains(err.Error(), "Duplicate") {
			return apperrors.CodeError(apperrors.CodeDevLogAlreadyLiked)
		}
		return fmt.Errorf("创建点赞记录失败: %w", err)
	}
	return nil
}

// DeleteLike 删除点赞记录
func (r *devLogLikeRepository) DeleteLike(ctx context.Context, logID, userID uint64) error {
	result := r.db.WithContext(ctx).
		Where("log_id = ? AND user_id = ?", logID, userID).
		Delete(&model.DevLogLike{})
	if result.Error != nil {
		return fmt.Errorf("删除点赞记录失败: %w", result.Error)
	}
	return nil
}

// HasLiked 判断用户是否已点赞
func (r *devLogLikeRepository) HasLiked(ctx context.Context, logID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DevLogLike{}).
		Where("log_id = ? AND user_id = ?", logID, userID).
		Count(&count).Error
	return count > 0, err
}

// CountLikes 统计日志点赞总数
func (r *devLogLikeRepository) CountLikes(ctx context.Context, logID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DevLogLike{}).
		Where("log_id = ?", logID).
		Count(&count).Error
	return count, err
}

// GetLikedLogIDs 批量查询用户对指定日志列表中已点赞的 ID 集合
func (r *devLogLikeRepository) GetLikedLogIDs(ctx context.Context, userID uint64, logIDs []uint64) (map[uint64]bool, error) {
	if len(logIDs) == 0 {
		return map[uint64]bool{}, nil
	}
	var likedIDs []uint64
	err := r.db.WithContext(ctx).
		Model(&model.DevLogLike{}).
		Select("log_id").
		Where("user_id = ? AND log_id IN ?", userID, logIDs).
		Pluck("log_id", &likedIDs).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint64]bool, len(likedIDs))
	for _, id := range likedIDs {
		result[id] = true
	}
	return result, nil
}

// CreateCollect 创建收藏记录
func (r *devLogCollectRepository) CreateCollect(ctx context.Context, collect *model.DevLogCollect) error {
	if err := r.db.WithContext(ctx).Create(collect).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") ||
			strings.Contains(err.Error(), "Duplicate") {
			return apperrors.CodeError(apperrors.CodeDevLogAlreadyCollected)
		}
		return fmt.Errorf("创建收藏记录失败: %w", err)
	}
	return nil
}

// DeleteCollect 删除收藏记录
func (r *devLogCollectRepository) DeleteCollect(ctx context.Context, logID, userID uint64) error {
	result := r.db.WithContext(ctx).
		Where("log_id = ? AND user_id = ?", logID, userID).
		Delete(&model.DevLogCollect{})
	if result.Error != nil {
		return fmt.Errorf("删除收藏记录失败: %w", result.Error)
	}
	return nil
}

// HasCollected 判断用户是否已收藏
func (r *devLogCollectRepository) HasCollected(ctx context.Context, logID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DevLogCollect{}).
		Where("log_id = ? AND user_id = ?", logID, userID).
		Count(&count).Error
	return count > 0, err
}

// ===========================
// DevLogCommentLikeRepository 实现
// ===========================

// CreateCommentLike 创建评论点赞记录
func (r *devLogCommentLikeRepository) CreateCommentLike(ctx context.Context, like *model.DevLogCommentLike) error {
	if err := r.db.WithContext(ctx).Create(like).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") ||
			strings.Contains(err.Error(), "Duplicate") {
			return apperrors.CodeError(apperrors.CodeDevLogCommentAlreadyLiked)
		}
		return fmt.Errorf("创建评论点赞记录失败: %w", err)
	}
	return nil
}

// DeleteCommentLike 删除评论点赞记录
func (r *devLogCommentLikeRepository) DeleteCommentLike(ctx context.Context, commentID, userID uint64) error {
	result := r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Delete(&model.DevLogCommentLike{})
	if result.Error != nil {
		return fmt.Errorf("删除评论点赞记录失败: %w", result.Error)
	}
	return nil
}

// HasCommentLiked 判断用户是否已对评论点赞
func (r *devLogCommentLikeRepository) HasCommentLiked(ctx context.Context, commentID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DevLogCommentLike{}).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetLikedCommentIDs 批量判断用户对哪些评论已点赞，返回 map[commentID]true
func (r *devLogCommentLikeRepository) GetLikedCommentIDs(ctx context.Context, userID uint64, commentIDs []uint64) (map[uint64]bool, error) {
	if len(commentIDs) == 0 {
		return map[uint64]bool{}, nil
	}
	var liked []uint64
	err := r.db.WithContext(ctx).Model(&model.DevLogCommentLike{}).
		Where("user_id = ? AND comment_id IN ?", userID, commentIDs).
		Pluck("comment_id", &liked).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint64]bool, len(liked))
	for _, id := range liked {
		result[id] = true
	}
	return result, nil
}

// ListCollectedLogsByUserID 分页查询用户收藏的日志 ID 列表
func (r *devLogCollectRepository) ListCollectedLogsByUserID(ctx context.Context, userID uint64, offset, limit int) ([]uint64, int64, error) {
	var total int64
	var collects []model.DevLogCollect

	if err := r.db.WithContext(ctx).Model(&model.DevLogCollect{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计收藏日志数失败: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&model.DevLogCollect{}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&collects).Error; err != nil {
		return nil, 0, fmt.Errorf("查询收藏日志失败: %w", err)
	}

	logIDs := make([]uint64, 0, len(collects))
	for _, c := range collects {
		logIDs = append(logIDs, c.LogID)
	}
	return logIDs, total, nil
}

// GetCollectedLogIDs 批量查询用户对指定日志列表中已收藏的 ID 集合
func (r *devLogCollectRepository) GetCollectedLogIDs(ctx context.Context, userID uint64, logIDs []uint64) (map[uint64]bool, error) {
	if len(logIDs) == 0 {
		return map[uint64]bool{}, nil
	}
	var collectedIDs []uint64
	err := r.db.WithContext(ctx).
		Model(&model.DevLogCollect{}).
		Select("log_id").
		Where("user_id = ? AND log_id IN ?", userID, logIDs).
		Pluck("log_id", &collectedIDs).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint64]bool, len(collectedIDs))
	for _, id := range collectedIDs {
		result[id] = true
	}
	return result, nil
}
func (r *devLogRepository) ListLogsByAuthorID(ctx context.Context, authorID uint64, offset, limit int) ([]*model.DevLog, int64, error) {
	var total int64
	var logs []*model.DevLog

	query := r.db.WithContext(ctx).Model(&model.DevLog{}).
		Where("author_id = ? AND status = 'published' AND deleted_at IS NULL", authorID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计作者日志数失败: %w", err)
	}

	if err := query.Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询作者日志列表失败: %w", err)
	}
	return logs, total, nil
}
