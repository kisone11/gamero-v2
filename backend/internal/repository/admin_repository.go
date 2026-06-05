// Package repository 提供管理后台专用的数据库访问层。
// 所有管理员操作均通过此文件中的接口和实现完成，
// 包含用户管理、内容审核、认证审核、话题管理、提现审核和数据统计。
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/gamero/gamero/internal/model"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"gorm.io/gorm"
)

// ===========================
// 统计数据结构体
// ===========================

// PlatformStats 平台概览统计数据
type PlatformStats struct {
	TotalUsers       int64 `json:"total_users"`        // 总用户数
	ActiveUsers      int64 `json:"active_users"`       // 活跃用户数（最近30天有活动）
	TotalProjects    int64 `json:"total_projects"`     // 总项目数
	TotalLogs        int64 `json:"total_logs"`         // 总日志数
	TotalPosts       int64 `json:"total_posts"`        // 总帖子数
	NewUsersToday    int64 `json:"new_users_today"`    // 今日新增用户数
	NewProjectsToday int64 `json:"new_projects_today"` // 今日新增项目数
}

// DailyStats 每日数据统计（用于图表）
type DailyStats struct {
	Date        string `json:"date"`         // 日期，格式 YYYY-MM-DD
	NewUsers    int64  `json:"new_users"`    // 当日新增用户数
	ActiveUsers int64  `json:"active_users"` // 当日活跃用户数
	NewProjects int64  `json:"new_projects"` // 当日新增项目数
	NewLogs     int64  `json:"new_logs"`     // 当日新增日志数
	NewPosts    int64  `json:"new_posts"`    // 当日新增帖子数
}

// ===========================
// 接口定义
// ===========================

// AdminRepository 管理后台仓储接口
// 包含用户管理、内容审核、认证审核、话题管理、提现审核和数据统计
type AdminRepository interface {
	// ===== 用户管理 =====

	// ListUsers 分页查询用户列表，支持关键词搜索（用户名/昵称/邮箱），按注册时间倒序
	ListUsers(ctx context.Context, keyword string, offset, limit int) ([]*model.User, int64, error)
	// GetUserByID 根据 ID 查询用户
	GetUserByID(ctx context.Context, id uint64) (*model.User, error)
	// GetUsersByIDs 批量查询用户，返回 map[id]*User
	GetUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]*model.User, error)
	// BanUser 封禁或解封用户（设置 status 字段；bannedUntil 为 nil 时永久封号）
	BanUser(ctx context.Context, userID uint64, banned bool, bannedUntil *time.Time) error
	// UnbanExpiredUsers 解封所有 banned_until <= now 的用户，返回解封数量
	UnbanExpiredUsers(ctx context.Context) (int64, error)
	// AdminDeleteUser 软删除用户（设置 deleted_at）
	AdminDeleteUser(ctx context.Context, userID uint64) error

	// ===== 内容审核 =====

	// ListReports 分页查询举报列表，支持按 target_type 和 status 筛选
	ListReports(ctx context.Context, targetType, status string, offset, limit int) ([]*model.Report, int64, error)
	// GetReportByID 按 ID 查询单条举报记录
	GetReportByID(ctx context.Context, reportID uint64) (*model.Report, error)
	// HandleReport 处理举报（更新 status 和 admin_note）
	HandleReport(ctx context.Context, reportID uint64, status, note string) error
	// AdminDeletePost 强制软删除帖子（不校验作者）
	AdminDeletePost(ctx context.Context, postID uint64) error
	// AdminDeleteComment 强制软删除评论
	// commentType: "post" 删除帖子评论，"log" 删除日志评论
	AdminDeleteComment(ctx context.Context, commentID uint64, commentType string) error

	// ===== 话题管理 =====

	// AdminCreateTopic 创建话题
	AdminCreateTopic(ctx context.Context, topic *model.Topic) error
	// AdminUpdateTopic 更新话题指定字段
	AdminUpdateTopic(ctx context.Context, topicID uint64, updates map[string]interface{}) error
	// AdminDeleteTopic 软删除话题（设置 deleted_at，不影响已关联帖子）
	AdminDeleteTopic(ctx context.Context, topicID uint64) error
	// GetTopicByIDForAdmin 查询话题（包含已软删除的，供管理员确认操作前使用）
	GetTopicByIDForAdmin(ctx context.Context, topicID uint64) (*model.Topic, error)

	// ===== 数据统计 =====

	// GetPlatformStats 获取平台概览统计数据
	GetPlatformStats(ctx context.Context) (*PlatformStats, error)
	// GetDailyStats 获取过去 N 天每天的新增用户数和新增项目数
	GetDailyStats(ctx context.Context, days int) ([]DailyStats, error)

	// ===== 审计日志 =====

	// CreateAuditLog 创建管理员操作审计日志
	CreateAuditLog(ctx context.Context, log *model.AdminAuditLog) error
	// ListAuditLogs 分页查询审计日志，adminID=0 表示查询所有管理员的日志
	ListAuditLogs(ctx context.Context, adminID uint64, offset, limit int) ([]*model.AdminAuditLog, int64, error)

	// ===== 项目管理 =====

	// BanProject 下架/恢复项目（设置 is_banned + ban_reason）
	BanProject(ctx context.Context, projectID uint64, banned bool, reason string) error
	// GetProjectByIDForAdmin 查询项目（忽略软删除，供管理员使用）
	GetProjectByIDForAdmin(ctx context.Context, projectID uint64) (*model.Project, error)
	// ListProjects 分页查询项目列表
	ListProjects(ctx context.Context, keyword string, offset, limit int) ([]*model.Project, int64, error)
	// ListPosts 分页查询帖子列表
	ListPosts(ctx context.Context, keyword string, offset, limit int) ([]*model.Post, int64, error)
	// ListLogs 分页查询开发日志列表
	ListLogs(ctx context.Context, keyword string, offset, limit int) ([]*model.DevLog, int64, error)
	// ListTopics 查询话题列表
	ListTopics(ctx context.Context) ([]*model.Topic, error)

	// DB 暴露底层 *gorm.DB，供 service 层构造跨表事务
	DB() *gorm.DB
}

// ===========================
// 实现
// ===========================

// adminRepository AdminRepository 的 GORM 实现
type adminRepository struct {
	db *gorm.DB
}

// NewAdminRepository 创建 AdminRepository 实例
func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

// ===== 用户管理实现 =====

// ListUsers 分页查询用户列表
// 支持关键词搜索（用户名/昵称/邮箱），按注册时间倒序
func (r *adminRepository) ListUsers(ctx context.Context, keyword string, offset, limit int) ([]*model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("deleted_at IS NULL")

	// 关键词过滤：匹配用户名、昵称、邮箱
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", like, like, like)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，按注册时间倒序
	var users []*model.User
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetUserByID 根据 ID 查询用户（包含已软删除的用户不在此处，仅查询正常用户）
func (r *adminRepository) GetUserByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUsersByIDs 批量查询用户，返回 map[id]*User
func (r *adminRepository) GetUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]*model.User, error) {
	if len(ids) == 0 {
		return map[uint64]*model.User{}, nil
	}
	var users []*model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	result := make(map[uint64]*model.User, len(users))
	for _, u := range users {
		result[u.ID] = u
	}
	return result, nil
}

// BanUser 封禁或解封用户（更新 status + banned_until）
func (r *adminRepository) BanUser(ctx context.Context, userID uint64, banned bool, bannedUntil *time.Time) error {
	updates := map[string]interface{}{"banned_until": nil, "is_banned": false}
	if banned {
		updates["status"] = model.UserStatusDisabled
		updates["banned_until"] = bannedUntil
		updates["is_banned"] = true
	} else {
		updates["status"] = model.UserStatusActive
	}
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

// UnbanExpiredUsers 解封所有封号到期的用户（status=0 且 banned_until <= now）
func (r *adminRepository) UnbanExpiredUsers(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("status = ? AND banned_until IS NOT NULL AND banned_until <= ?",
			model.UserStatusDisabled, time.Now()).
		Updates(map[string]interface{}{
			"status":       model.UserStatusActive,
			"banned_until": nil,
		})
	return result.RowsAffected, result.Error
}

// AdminDeleteUser 软删除用户（通过 GORM 的 DeletedAt 机制）
// User 模型若使用标准 gorm.Model 会有 DeletedAt，否则直接更新 deleted_at 列
func (r *adminRepository) AdminDeleteUser(ctx context.Context, userID uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("deleted_at", now).Error
}

// ===== 内容审核实现 =====

// ListReports 分页查询举报列表
func (r *adminRepository) ListReports(ctx context.Context, targetType, status string, offset, limit int) ([]*model.Report, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Report{})

	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reports []*model.Report
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&reports).Error; err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

// GetReportByID 查询单条举报记录
func (r *adminRepository) GetReportByID(ctx context.Context, reportID uint64) (*model.Report, error) {
	var report model.Report
	if err := r.db.WithContext(ctx).First(&report, reportID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(apperrors.CodeNotFound, "举报记录不存在")
		}
		return nil, err
	}
	return &report, nil
}

// HandleReport 处理举报，更新状态和管理员备注
func (r *adminRepository) HandleReport(ctx context.Context, reportID uint64, status, note string) error {
	return r.db.WithContext(ctx).Model(&model.Report{}).
		Where("id = ?", reportID).
		Updates(map[string]interface{}{
			"status":     status,
			"admin_note": note,
		}).Error
}

// AdminDeletePost 强制软删除帖子
func (r *adminRepository) AdminDeletePost(ctx context.Context, postID uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND deleted_at IS NULL", postID).
		Update("deleted_at", now).Error
}

// AdminDeleteComment 强制软删除评论
// commentType: "post" 操作 post_comments 表，"log" 操作 dev_log_comments 表
func (r *adminRepository) AdminDeleteComment(ctx context.Context, commentID uint64, commentType string) error {
	now := time.Now()
	var tableName string
	switch commentType {
	case "log":
		tableName = "dev_log_comments"
	default:
		// 默认为帖子评论
		tableName = "post_comments"
	}
	return r.db.WithContext(ctx).Table(tableName).
		Where("id = ? AND deleted_at IS NULL", commentID).
		Update("deleted_at", now).Error
}

// ===== 话题管理实现 =====

// AdminCreateTopic 创建话题
func (r *adminRepository) AdminCreateTopic(ctx context.Context, topic *model.Topic) error {
	return r.db.WithContext(ctx).Create(topic).Error
}

// AdminUpdateTopic 更新话题指定字段
func (r *adminRepository) AdminUpdateTopic(ctx context.Context, topicID uint64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Topic{}).
		Where("id = ?", topicID).
		Updates(updates).Error
}

// AdminDeleteTopic 删除话题
func (r *adminRepository) AdminDeleteTopic(ctx context.Context, topicID uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Topic{}, topicID).Error
}

// GetTopicByIDForAdmin 查询话题（管理员视角，包含已软删除的）
func (r *adminRepository) GetTopicByIDForAdmin(ctx context.Context, topicID uint64) (*model.Topic, error) {
	var topic model.Topic
	// 使用原始 SQL 条件，跳过 GORM 的软删除过滤
	if err := r.db.WithContext(ctx).Unscoped().First(&topic, topicID).Error; err != nil {
		return nil, err
	}
	return &topic, nil
}

// ===== 数据统计实现 =====

// GetPlatformStats 获取平台概览统计数据
func (r *adminRepository) GetPlatformStats(ctx context.Context) (*PlatformStats, error) {
	stats := &PlatformStats{}
	today := time.Now().Truncate(24 * time.Hour)
	thirtyDaysAgo := today.AddDate(0, 0, -30)

	// 总用户数（User 模型不使用软删除，无需 deleted_at 条件）
	if err := r.db.WithContext(ctx).Model(&model.User{}).Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}

	// 活跃用户数（最近30天有登录或创建内容）
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("updated_at >= ?", thirtyDaysAgo).
		Count(&stats.ActiveUsers).Error; err != nil {
		return nil, err
	}

	// 总项目数
	if err := r.db.WithContext(ctx).Table("projects").Where("deleted_at IS NULL").Count(&stats.TotalProjects).Error; err != nil {
		return nil, err
	}

	// 总日志数
	if err := r.db.WithContext(ctx).Model(&model.DevLog{}).Where("deleted_at IS NULL").Count(&stats.TotalLogs).Error; err != nil {
		return nil, err
	}

	// 总帖子数
	if err := r.db.WithContext(ctx).Model(&model.Post{}).Where("deleted_at IS NULL").Count(&stats.TotalPosts).Error; err != nil {
		return nil, err
	}

	// 今日新增用户数
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("created_at >= ?", today).Count(&stats.NewUsersToday).Error; err != nil {
		return nil, err
	}

	// 今日新增项目数
	if err := r.db.WithContext(ctx).Table("projects").Where("created_at >= ? AND deleted_at IS NULL", today).Count(&stats.NewProjectsToday).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetDailyStats 获取过去 N 天每天的新增用户数和新增项目数
// 结果按日期从旧到新排序，返回长度 = days
func (r *adminRepository) GetDailyStats(ctx context.Context, days int) ([]DailyStats, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	// 生成过去 days 天的日期列表（从今天往前推）
	now := time.Now()
	result := make([]DailyStats, days)
	for i := 0; i < days; i++ {
		// 从最旧的日期开始，index 0 = days-1 天前
		dayOffset := days - 1 - i
		date := now.AddDate(0, 0, -dayOffset)
		result[i] = DailyStats{
			Date: date.Format("2006-01-02"),
		}
	}

	// 查询过去 days 天每天新增用户数
	type dailyCount struct {
		Date  string
		Count int64
	}

	startDate := now.AddDate(0, 0, -(days - 1)).Truncate(24 * time.Hour)

	var userCounts []dailyCount
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("DATE(created_at)").
		Scan(&userCounts).Error; err != nil {
		return nil, err
	}

	// 构建日期 -> 用户数映射
	userCountMap := make(map[string]int64)
	for _, uc := range userCounts {
		userCountMap[uc.Date] = uc.Count
	}

	// 查询过去 days 天每天新增项目数
	var projectCounts []dailyCount
	if err := r.db.WithContext(ctx).Table("projects").
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", startDate).
		Group("DATE(created_at)").
		Scan(&projectCounts).Error; err != nil {
		return nil, err
	}

	// 构建日期 -> 项目数映射
	projectCountMap := make(map[string]int64)
	for _, pc := range projectCounts {
		projectCountMap[pc.Date] = pc.Count
	}

	var logCounts []dailyCount
	if err := r.db.WithContext(ctx).Model(&model.DevLog{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", startDate).
		Group("DATE(created_at)").
		Scan(&logCounts).Error; err != nil {
		return nil, err
	}
	logCountMap := make(map[string]int64)
	for _, lc := range logCounts {
		logCountMap[lc.Date] = lc.Count
	}

	var postCounts []dailyCount
	if err := r.db.WithContext(ctx).Model(&model.Post{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", startDate).
		Group("DATE(created_at)").
		Scan(&postCounts).Error; err != nil {
		return nil, err
	}
	postCountMap := make(map[string]int64)
	for _, pc := range postCounts {
		postCountMap[pc.Date] = pc.Count
	}

	var activeCounts []dailyCount
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Select("DATE(updated_at) as date, COUNT(*) as count").
		Where("updated_at >= ? AND deleted_at IS NULL", startDate).
		Group("DATE(updated_at)").
		Scan(&activeCounts).Error; err != nil {
		return nil, err
	}
	activeCountMap := make(map[string]int64)
	for _, ac := range activeCounts {
		activeCountMap[ac.Date] = ac.Count
	}

	// 填充结果
	for i := range result {
		date := result[i].Date
		result[i].NewUsers = userCountMap[date]
		result[i].NewProjects = projectCountMap[date]
		result[i].NewLogs = logCountMap[date]
		result[i].NewPosts = postCountMap[date]
		result[i].ActiveUsers = activeCountMap[date]
	}

	return result, nil
}

// ===== 审计日志实现 =====

// CreateAuditLog 创建管理员操作审计日志
func (r *adminRepository) CreateAuditLog(ctx context.Context, log *model.AdminAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// ListAuditLogs 分页查询审计日志
// adminID=0 表示查询所有管理员的日志，否则只查询指定管理员的日志
func (r *adminRepository) ListAuditLogs(ctx context.Context, adminID uint64, offset, limit int) ([]*model.AdminAuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AdminAuditLog{})

	if adminID != 0 {
		query = query.Where("admin_id = ?", adminID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []*model.AdminAuditLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// BanProject 下架/恢复项目
func (r *adminRepository) BanProject(ctx context.Context, projectID uint64, banned bool, reason string) error {
	updates := map[string]interface{}{
		"is_banned":  banned,
		"ban_reason": reason,
	}
	result := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("id = ?", projectID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeNotFound)
	}
	return nil
}

// GetProjectByIDForAdmin 查询项目（包含软删除记录，供管理员使用）
func (r *adminRepository) GetProjectByIDForAdmin(ctx context.Context, projectID uint64) (*model.Project, error) {
	var p model.Project
	err := r.db.WithContext(ctx).
		Unscoped().
		Where("id = ?", projectID).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeNotFound)
		}
		return nil, err
	}
	return &p, nil
}

// DB 返回底层 *gorm.DB，供 service 层构造跨表事务
func (r *adminRepository) DB() *gorm.DB {
	return r.db
}

// ListProjects 分页查询项目列表
func (r *adminRepository) ListProjects(ctx context.Context, keyword string, offset, limit int) ([]*model.Project, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Project{}).Where("deleted_at IS NULL")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR slug LIKE ? OR description LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var projects []*model.Project
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&projects).Error; err != nil {
		return nil, 0, err
	}
	return projects, total, nil
}

// ListPosts 分页查询帖子列表
func (r *adminRepository) ListPosts(ctx context.Context, keyword string, offset, limit int) ([]*model.Post, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Post{}).Where("deleted_at IS NULL")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var posts []*model.Post
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error; err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

// ListLogs 分页查询开发日志列表
func (r *adminRepository) ListLogs(ctx context.Context, keyword string, offset, limit int) ([]*model.DevLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.DevLog{}).Where("deleted_at IS NULL")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []*model.DevLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// ListTopics 查询话题列表
func (r *adminRepository) ListTopics(ctx context.Context) ([]*model.Topic, error) {
	var topics []*model.Topic
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&topics).Error; err != nil {
		return nil, err
	}
	return topics, nil
}
