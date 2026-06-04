// Package repository 提供创作者数据统计的数据库访问层。
// 采用接口 + 实现的模式，便于单元测试时替换 Mock 实现。
// 本文件包含以下能力：
//   - 用户维度：全量汇总统计、区间每日统计、单日聚合、写入
//   - 项目维度：全量汇总统计、区间每日统计、单日聚合、写入
//   - 活跃用户/项目列表查询（定时聚合任务使用）
package repository

import (
	"context"
	"time"

	"github.com/gamero/gamero/internal/model"
	"gorm.io/gorm"
)

// CreatorTotalStats 创作者全量汇总统计（全时实时查询）
type CreatorTotalStats struct {
	TotalProjects      int64 `json:"total_projects"`      // 项目数（owner）
	TotalLogs          int64 `json:"total_logs"`          // 日志数（author）
	TotalPosts         int64 `json:"total_posts"`         // 帖子数（author）
	TotalFollowers     int64 `json:"total_followers"`     // 粉丝数
	TotalFollowing     int64 `json:"total_following"`     // 关注数
	TotalLikesReceived int64 `json:"total_likes_received"` // 获赞总数
	TotalViews         int64 `json:"total_views"`         // 总浏览量
}

// ProjectSummaryStats 项目汇总统计（全时聚合）
type ProjectSummaryStats struct {
	TotalViews     int64 `json:"total_views"`
	TotalLikes     int64 `json:"total_likes"`
	TotalFollowers int64 `json:"total_followers"`
}

// StatsRepository 创作者数据统计仓储接口
type StatsRepository interface {
	// ===== 用户维度 =====

	// GetUserDailyStats 获取用户指定日期范围内的每日统计，按日期升序。
	GetUserDailyStats(ctx context.Context, userID uint64, from, to time.Time) ([]*model.DailyStats, error)
	// GetUserTotalStats 获取用户全量汇总统计（跨所有业务表实时计算）。
	GetUserTotalStats(ctx context.Context, userID uint64) (*CreatorTotalStats, error)
	// ListActiveUserIDs 获取指定日期有活跃记录的用户 ID 列表，供定时聚合使用。
	ListActiveUserIDs(ctx context.Context, date time.Time) ([]uint64, error)
	// AggregateUserDailyStats 从各业务表实时聚合指定用户在指定日期的统计数据。
	AggregateUserDailyStats(ctx context.Context, userID uint64, date time.Time) (*model.DailyStats, error)
	// UpsertDailyStats 写入或更新用户每日统计（按 user_id + date 唯一约束）。
	UpsertDailyStats(ctx context.Context, stats *model.DailyStats) error

	// ===== 项目维度 =====

	// GetProjectDailyStats 获取项目指定日期范围内的每日统计，按日期升序。
	GetProjectDailyStats(ctx context.Context, projectID uint64, from, to time.Time) ([]*model.ProjectStats, error)
	// GetProjectSummaryStats 获取项目全量汇总统计。
	GetProjectSummaryStats(ctx context.Context, projectID uint64) (*ProjectSummaryStats, error)
	// ListActiveProjectIDs 获取指定日期有活跃记录的项目 ID 列表，供定时聚合使用。
	ListActiveProjectIDs(ctx context.Context, date time.Time) ([]uint64, error)
	// AggregateProjectDailyStats 从各业务表实时聚合指定项目在指定日期的统计数据。
	AggregateProjectDailyStats(ctx context.Context, projectID uint64, date time.Time) (*model.ProjectStats, error)
	// UpsertProjectStats 写入或更新项目每日统计（按 project_id + date 唯一约束）。
	UpsertProjectStats(ctx context.Context, stats *model.ProjectStats) error
}

// statsRepository 是 StatsRepository 接口的具体实现
type statsRepository struct {
	db *gorm.DB
}

// NewStatsRepository 创建 statsRepository 实例
func NewStatsRepository(db *gorm.DB) StatsRepository {
	return &statsRepository{db: db}
}

// ==================== 用户维度实现 ====================

// GetUserDailyStats 查询用户指定日期范围内的每日统计，按日期升序
func (r *statsRepository) GetUserDailyStats(ctx context.Context, userID uint64, from, to time.Time) ([]*model.DailyStats, error) {
	var stats []*model.DailyStats
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND date >= ? AND date <= ?", userID, from, to).
		Order("date ASC").
		Find(&stats).Error
	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []*model.DailyStats{}
	}
	return stats, nil
}

// GetUserTotalStats 查询用户全量汇总统计，实时从各业务表计算
func (r *statsRepository) GetUserTotalStats(ctx context.Context, userID uint64) (*CreatorTotalStats, error) {
	var total CreatorTotalStats

	// 项目数：projects 表使用 owner_id（修复：原为 author_id，与 projects 表字段不匹配）
	if err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("owner_id = ? AND deleted_at IS NULL", userID).
		Count(&total.TotalProjects).Error; err != nil {
		return nil, err
	}

	// 日志数：dev_logs 表使用 author_id
	if err := r.db.WithContext(ctx).
		Model(&model.DevLog{}).
		Where("author_id = ? AND deleted_at IS NULL", userID).
		Count(&total.TotalLogs).Error; err != nil {
		return nil, err
	}

	// 帖子数：posts 表使用 author_id
	if err := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("author_id = ? AND deleted_at IS NULL", userID).
		Count(&total.TotalPosts).Error; err != nil {
		return nil, err
	}

	// 粉丝数：user_follows 表 followed_id
	if err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("followed_id = ?", userID).
		Count(&total.TotalFollowers).Error; err != nil {
		return nil, err
	}

	// 关注数：user_follows 表 follower_id
	if err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("follower_id = ?", userID).
		Count(&total.TotalFollowing).Error; err != nil {
		return nil, err
	}

	// 获赞总数：统计用户所发帖子的点赞数
	if err := r.db.WithContext(ctx).
		Model(&model.PostLike{}).
		Joins("JOIN posts ON posts.id = post_likes.post_id").
		Where("posts.author_id = ? AND posts.deleted_at IS NULL", userID).
		Count(&total.TotalLikesReceived).Error; err != nil {
		return nil, err
	}

	// 总浏览量：用户所有帖子的 view_count 累计和
	if err := r.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("author_id = ? AND deleted_at IS NULL", userID).
		Select("COALESCE(SUM(view_count), 0)").
		Scan(&total.TotalViews).Error; err != nil {
		return nil, err
	}

	return &total, nil
}

// ListActiveUserIDs 获取指定日期有活跃记录的用户 ID 列表
// 活跃定义：当天有新粉丝、被评论、被点赞、日志被点赞，或前一天有历史统计记录
func (r *statsRepository) ListActiveUserIDs(ctx context.Context, date time.Time) ([]uint64, error) {
	dateStr := date.Format("2006-01-02")
	prevDateStr := date.AddDate(0, 0, -1).Format("2006-01-02")

	query := `
		SELECT DISTINCT user_id FROM (
			SELECT followed_id AS user_id FROM user_follows WHERE DATE(created_at) = ?
			UNION
			SELECT p.author_id FROM post_comments pc
				JOIN posts p ON p.id = pc.post_id
				WHERE DATE(pc.created_at) = ? AND p.deleted_at IS NULL
			UNION
			SELECT p.author_id FROM post_likes pl
				JOIN posts p ON p.id = pl.post_id
				WHERE DATE(pl.created_at) = ? AND p.deleted_at IS NULL
			UNION
			SELECT dl.author_id FROM dev_log_likes dll
				JOIN dev_logs dl ON dl.id = dll.log_id
				WHERE DATE(dll.created_at) = ? AND dl.deleted_at IS NULL
			UNION
			SELECT user_id FROM daily_stats WHERE date = ?
		) AS active
	`

	var ids []uint64
	err := r.db.WithContext(ctx).Raw(query, dateStr, dateStr, dateStr, dateStr, prevDateStr).
		Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint64{}
	}
	return ids, nil
}

// AggregateUserDailyStats 从各业务表聚合指定用户在指定日期的统计数据
// 注意：PostViews 和 LogViews 由 Redis HyperLogLog 单独统计，此处不聚合
func (r *statsRepository) AggregateUserDailyStats(ctx context.Context, userID uint64, date time.Time) (*model.DailyStats, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	stats := &model.DailyStats{
		UserID: userID,
		Date:   startOfDay,
	}

	// 帖子获赞数：当天 post_likes 中被赞帖子的作者为该用户
	if err := r.db.WithContext(ctx).
		Model(&model.PostLike{}).
		Joins("JOIN posts ON posts.id = post_likes.post_id").
		Where("posts.author_id = ? AND posts.deleted_at IS NULL", userID).
		Where("post_likes.created_at >= ? AND post_likes.created_at < ?", startOfDay, endOfDay).
		Count(&stats.PostLikes).Error; err != nil {
		return nil, err
	}

	// 帖子评论数：当天 post_comments 中评论帖子的作者为该用户
	if err := r.db.WithContext(ctx).
		Model(&model.PostComment{}).
		Joins("JOIN posts ON posts.id = post_comments.post_id").
		Where("posts.author_id = ? AND posts.deleted_at IS NULL", userID).
		Where("post_comments.created_at >= ? AND post_comments.created_at < ?", startOfDay, endOfDay).
		Count(&stats.PostComments).Error; err != nil {
		return nil, err
	}

	// 日志获赞数：当天 dev_log_likes 中被赞日志的作者为该用户
	if err := r.db.WithContext(ctx).
		Model(&model.DevLogLike{}).
		Joins("JOIN dev_logs ON dev_logs.id = dev_log_likes.log_id").
		Where("dev_logs.author_id = ? AND dev_logs.deleted_at IS NULL", userID).
		Where("dev_log_likes.created_at >= ? AND dev_log_likes.created_at < ?", startOfDay, endOfDay).
		Count(&stats.LogLikes).Error; err != nil {
		return nil, err
	}

	// 新增粉丝数：当天新增的 user_follows 记录中 followed_id 为该用户
	if err := r.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("followed_id = ?", userID).
		Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
		Count(&stats.NewFollowers).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// UpsertDailyStats 写入或更新用户每日统计记录
// 使用 FirstOrCreate + Assign 实现 upsert 语义（按 user_id + date 唯一约束）
func (r *statsRepository) UpsertDailyStats(ctx context.Context, stats *model.DailyStats) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND date = ?", stats.UserID, stats.Date).
		Assign(map[string]interface{}{
			"post_views":    stats.PostViews,
			"post_likes":    stats.PostLikes,
			"post_comments": stats.PostComments,
			"log_views":     stats.LogViews,
			"log_likes":     stats.LogLikes,
			"new_followers": stats.NewFollowers,
		}).
		FirstOrCreate(stats).Error
}

// ==================== 项目维度实现 ====================

// GetProjectDailyStats 查询项目指定日期范围内的每日统计，按日期升序
func (r *statsRepository) GetProjectDailyStats(ctx context.Context, projectID uint64, from, to time.Time) ([]*model.ProjectStats, error) {
	var stats []*model.ProjectStats
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND date >= ? AND date <= ?", projectID, from, to).
		Order("date ASC").
		Find(&stats).Error
	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []*model.ProjectStats{}
	}
	return stats, nil
}

// GetProjectSummaryStats 获取项目全量汇总统计
// 从 project_stats 表中聚合 views/likes，followers 取最新日期的累计值
func (r *statsRepository) GetProjectSummaryStats(ctx context.Context, projectID uint64) (*ProjectSummaryStats, error) {
	var summary ProjectSummaryStats

	err := r.db.WithContext(ctx).
		Model(&model.ProjectStats{}).
		Where("project_id = ?", projectID).
		Select(`
			COALESCE(SUM(views), 0) AS total_views,
			COALESCE(SUM(likes), 0) AS total_likes,
			COALESCE(MAX(followers), 0) AS total_followers
		`).
		Scan(&summary).Error
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// ListActiveProjectIDs 获取指定日期有活跃记录的项目 ID 列表
// 活跃定义：当天有新日志、日志被点赞、新增关注，或前一天有历史统计记录
func (r *statsRepository) ListActiveProjectIDs(ctx context.Context, date time.Time) ([]uint64, error) {
	dateStr := date.Format("2006-01-02")
	prevDateStr := date.AddDate(0, 0, -1).Format("2006-01-02")

	query := `
		SELECT DISTINCT project_id FROM (
			SELECT project_id FROM dev_logs
				WHERE DATE(created_at) = ? AND deleted_at IS NULL
			UNION
			SELECT dl.project_id FROM dev_log_likes dll
				JOIN dev_logs dl ON dl.id = dll.log_id
				WHERE DATE(dll.created_at) = ? AND dl.deleted_at IS NULL
			UNION
			SELECT project_id FROM project_follows
				WHERE DATE(created_at) = ?
			UNION
			SELECT project_id FROM project_stats
				WHERE date = ?
		) AS active
	`

	var ids []uint64
	err := r.db.WithContext(ctx).Raw(query, dateStr, dateStr, dateStr, prevDateStr).
		Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint64{}
	}
	return ids, nil
}

// AggregateProjectDailyStats 从各业务表聚合指定项目在指定日期的统计数据
// 注意：Views 由 Redis HyperLogLog 单独统计，此处暂不聚合
func (r *statsRepository) AggregateProjectDailyStats(ctx context.Context, projectID uint64, date time.Time) (*model.ProjectStats, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	stats := &model.ProjectStats{
		ProjectID: projectID,
		Date:      startOfDay,
	}

	// 点赞数：当天项目日志获得的点赞数
	if err := r.db.WithContext(ctx).
		Model(&model.DevLogLike{}).
		Joins("JOIN dev_logs ON dev_logs.id = dev_log_likes.log_id").
		Where("dev_logs.project_id = ? AND dev_logs.deleted_at IS NULL", projectID).
		Where("dev_log_likes.created_at >= ? AND dev_log_likes.created_at < ?", startOfDay, endOfDay).
		Count(&stats.Likes).Error; err != nil {
		return nil, err
	}

	// 粉丝数（累计）：截至当天结束的项目关注总数
	if err := r.db.WithContext(ctx).
		Model(&model.ProjectFollow{}).
		Where("project_id = ? AND created_at < ?", projectID, endOfDay).
		Count(&stats.Followers).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// UpsertProjectStats 写入或更新项目每日统计记录
// 使用 FirstOrCreate + Assign 实现 upsert 语义（按 project_id + date 唯一约束）
func (r *statsRepository) UpsertProjectStats(ctx context.Context, stats *model.ProjectStats) error {
	return r.db.WithContext(ctx).
		Where("project_id = ? AND date = ?", stats.ProjectID, stats.Date).
		Assign(map[string]interface{}{
			"views":     stats.Views,
			"likes":     stats.Likes,
			"followers": stats.Followers,
		}).
		FirstOrCreate(stats).Error
}
