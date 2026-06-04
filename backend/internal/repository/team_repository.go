// Package repository 提供组队系统的数据库访问层。
// 采用接口 + 实现的模式，封装所有对 Recruitment、RecruitmentApplication、
// CollaborationReview、Notification 表的操作。
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
// 接口定义
// ===========================

// ListRecruitmentsParams 招募列表查询参数
type ListRecruitmentsParams struct {
	Page            int                        // 页码
	PageSize        int                        // 每页大小
	Position        model.RecruitmentPosition  // 岗位筛选（空表示不筛选）
	CooperationType model.CooperationType      // 合作方式筛选（空表示不筛选）
	Status          model.RecruitmentStatus    // 状态筛选（空表示只显示open）
	ProjectID       uint64                     // 项目筛选（0表示不筛选）
	OwnerID         uint64                     // 发布者筛选（0表示不筛选）
	Keyword         string                     // 关键词（匹配标题或描述）
}

// ListNotificationsParams 通知列表查询参数
type ListNotificationsParams struct {
	UserID   uint64            // 用户 ID
	Page     int               // 页码
	PageSize int               // 每页大小
	IsRead   *bool             // 已读状态筛选（nil 表示不筛选）
	Type     model.NotificationType // 单个通知类型筛选（"" 表示不筛选，与 Types 互斥，Types 优先）
	Types    []model.NotificationType // 多类型筛选（IN 查询）；非空时忽略 Type 字段
}

// TeamRepository 组队系统仓储接口
// 整合 Recruitment、RecruitmentApplication、CollaborationReview、Notification 的所有操作
type TeamRepository interface {
	// ===== Recruitment 招募操作 =====

	// CreateRecruitment 创建招募信息
	CreateRecruitment(ctx context.Context, r *model.Recruitment) error
	// GetRecruitmentByID 根据 ID 查找招募（含状态检查）
	GetRecruitmentByID(ctx context.Context, id uint64) (*model.Recruitment, error)
	// UpdateRecruitment 更新招募信息（只更新指定字段）
	UpdateRecruitment(ctx context.Context, id uint64, updates map[string]interface{}) error
	// CloseRecruitment 关闭招募（将 status 设为 closed）
	CloseRecruitment(ctx context.Context, id uint64) error
	// ReopenRecruitment 重开招募（将 status 设为 open）
	ReopenRecruitment(ctx context.Context, id uint64) error
	// ListRecruitments 分页查询招募列表，支持多条件筛选
	ListRecruitments(ctx context.Context, params *ListRecruitmentsParams) ([]*model.Recruitment, int64, error)
	// ExpireRecruitments 批量将已超过 expire_at 的开放招募状态设为 expired
	ExpireRecruitments(ctx context.Context) (int64, error)
	// CloseExpiredRecruitments 关闭所有 expires_at 已过期的招募帖，返回关闭数量
	CloseExpiredRecruitments(ctx context.Context) (int64, error)
	// ListExpiringSoonRecruitments 查询 withinDays 天内即将过期的招募（status=open，expires_at 不为 nil）
	ListExpiringSoonRecruitments(ctx context.Context, withinDays int) ([]*model.Recruitment, error)

	// ===== RecruitmentApplication 申请操作 =====

	// CreateApplication 创建申请
	CreateApplication(ctx context.Context, app *model.RecruitmentApplication) error
	// GetApplicationByID 根据 ID 查找申请
	GetApplicationByID(ctx context.Context, id uint64) (*model.RecruitmentApplication, error)
	// GetApplicationsByRecruitmentID 查询招募下的所有申请（分页）
	GetApplicationsByRecruitmentID(ctx context.Context, recruitmentID uint64, page, pageSize int) ([]*model.RecruitmentApplication, int64, error)
	// GetApplicationsByApplicantID 查询用户提交的所有申请（分页）
	GetApplicationsByApplicantID(ctx context.Context, applicantID uint64, page, pageSize int) ([]*model.RecruitmentApplication, int64, error)
	// UpdateApplicationStatus 更新申请状态
	UpdateApplicationStatus(ctx context.Context, id uint64, status model.ApplicationStatus) error
	// HasApplied 判断用户是否已申请某招募（非撤回状态）
	HasApplied(ctx context.Context, applicantID, recruitmentID uint64) (bool, error)
	// GetApplicationByApplicantAndRecruitment 根据申请人和招募ID查找申请
	GetApplicationByApplicantAndRecruitment(ctx context.Context, applicantID, recruitmentID uint64) (*model.RecruitmentApplication, error)

	// ===== CollaborationReview 合作评价操作 =====

	// CreateReview 创建合作评价
	CreateReview(ctx context.Context, review *model.CollaborationReview) error
	// GetReviewByID 根据 ID 查找评价
	GetReviewByID(ctx context.Context, id uint64) (*model.CollaborationReview, error)
	// GetReviewsByRevieweeID 查询用户收到的所有评价（分页）
	GetReviewsByRevieweeID(ctx context.Context, revieweeID uint64, page, pageSize int) ([]*model.CollaborationReview, int64, error)
	// GetReviewByProjectAndUsers 根据项目和用户对查找评价
	GetReviewByProjectAndUsers(ctx context.Context, projectID, reviewerID, revieweeID uint64) (*model.CollaborationReview, error)
	// HasReviewed 判断评价人是否已对被评人在该项目中发表过评价
	HasReviewed(ctx context.Context, projectID, reviewerID, revieweeID uint64) (bool, error)
	// AddSupplement 补充评价说明（只能补充一次，已有补充则返回错误）
	AddSupplement(ctx context.Context, reviewID uint64, supplement string) error
	// GetAverageRating 获取用户的平均评分和评价总数
	GetAverageRating(ctx context.Context, revieweeID uint64) (float64, int64, error)

	// GetUserRecentEndorsements 查询用户收到的合作评价列表（reviewer != 自己），按时间倒序
	GetUserRecentEndorsements(ctx context.Context, userID uint64, limit int) ([]*model.CollaborationReview, error)

	// ===== Notification 通知操作 =====

	// CreateNotification 创建通知记录
	CreateNotification(ctx context.Context, n *model.Notification) error
	// CreateNotifications 批量创建通知记录（单次 INSERT，空列表时直接返回）
	CreateNotifications(ctx context.Context, ns []*model.Notification) error
	// GetNotificationByID 根据 ID 查找通知
	GetNotificationByID(ctx context.Context, id uint64) (*model.Notification, error)
	// MarkAsRead 将单条通知标记为已读
	MarkAsRead(ctx context.Context, id, userID uint64) error
	// MarkAllAsRead 将用户所有通知标记为已读
	MarkAllAsRead(ctx context.Context, userID uint64) error
	// GetNotifications 分页查询用户通知列表
	GetNotifications(ctx context.Context, params *ListNotificationsParams) ([]*model.Notification, int64, error)
	// CountUnread 统计用户未读通知数量
	CountUnread(ctx context.Context, userID uint64) (int64, error)
	// CountUnreadByTypes 按类型分组统计未读通知数量，返回 map[NotificationType]int64
	CountUnreadByTypes(ctx context.Context, userID uint64) (map[model.NotificationType]int64, error)
	// DeleteNotification 删除单条通知（需验证归属用户）
	DeleteNotification(ctx context.Context, userID, notificationID uint64) error
	// DeleteReadNotifications 删除用户所有已读通知
	DeleteReadNotifications(ctx context.Context, userID uint64) error
	// ClearNotifications 清空用户所有通知
	ClearNotifications(ctx context.Context, userID uint64) error
}

// ===========================
// 实现
// ===========================

// teamRepository 是 TeamRepository 接口的具体实现
type teamRepository struct {
	db *gorm.DB
}

// NewTeamRepository 创建 teamRepository 实例
func NewTeamRepository(db *gorm.DB) TeamRepository {
	return &teamRepository{db: db}
}

// ==================== Recruitment 招募操作实现 ====================

// CreateRecruitment 创建招募信息
func (r *teamRepository) CreateRecruitment(ctx context.Context, rec *model.Recruitment) error {
	if err := r.db.WithContext(ctx).Create(rec).Error; err != nil {
		return err
	}
	return nil
}

// GetRecruitmentByID 根据 ID 查找招募
func (r *teamRepository) GetRecruitmentByID(ctx context.Context, id uint64) (*model.Recruitment, error) {
	var rec model.Recruitment
	if err := r.db.WithContext(ctx).First(&rec, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeRecruitmentNotFound)
		}
		return nil, err
	}
	return &rec, nil
}

// UpdateRecruitment 更新招募信息
func (r *teamRepository) UpdateRecruitment(ctx context.Context, id uint64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&model.Recruitment{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// CloseRecruitment 将招募状态设为 closed
func (r *teamRepository) CloseRecruitment(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.Recruitment{}).
		Where("id = ?", id).
		Update("status", model.RecruitmentStatusClosed)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// ReopenRecruitment 将招募状态设为 open
func (r *teamRepository) ReopenRecruitment(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.Recruitment{}).
		Where("id = ?", id).
		Update("status", model.RecruitmentStatusOpen)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// ListRecruitments 分页查询招募列表
// 默认只显示 open 状态且未过期的招募，按发布时间倒序
func (r *teamRepository) ListRecruitments(ctx context.Context, params *ListRecruitmentsParams) ([]*model.Recruitment, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.Recruitment{})

	// 默认只查 open 状态且未过期的招募
	if params.Status != "" {
		db = db.Where("status = ?", params.Status)
	} else {
		db = db.Where("status = ? AND expire_at > ?", model.RecruitmentStatusOpen, time.Now())
	}

	// 按岗位筛选
	if params.Position != "" {
		db = db.Where("position = ?", params.Position)
	}

	// 按合作方式筛选
	if params.CooperationType != "" {
		db = db.Where("cooperation_type = ?", params.CooperationType)
	}

	// 按发布者筛选（查自己发布的招募）
	if params.OwnerID > 0 {
		db = db.Where("owner_id = ?", params.OwnerID)
	}

	// 关键词搜索（title 或 description）
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		db = db.Where("(title LIKE ? OR description LIKE ?)", like, like)
	}

	// 按项目筛选
	if params.ProjectID > 0 {
		db = db.Where("project_id = ?", params.ProjectID)
	}

	// 按发布者筛选
	if params.OwnerID > 0 {
		db = db.Where("owner_id = ?", params.OwnerID)
	}

	// 关键词搜索（描述）
	if params.Keyword != "" {
		like := "%" + params.Keyword + "%"
		db = db.Where("description LIKE ?", like)
	}

	// 统计总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，按发布时间倒序
	var list []*model.Recruitment
	offset := (params.Page - 1) * params.PageSize
	if err := db.Order("created_at DESC").
		Offset(offset).
		Limit(params.PageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// ExpireRecruitments 批量将过期招募设为 expired 状态
func (r *teamRepository) ExpireRecruitments(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&model.Recruitment{}).
		Where("status = ? AND expire_at <= ?", model.RecruitmentStatusOpen, time.Now()).
		Update("status", model.RecruitmentStatusExpired)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// CloseExpiredRecruitments 关闭所有 expires_at 已过期的招募帖
func (r *teamRepository) CloseExpiredRecruitments(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.Recruitment{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?",
			model.RecruitmentStatusOpen, time.Now()).
		Update("status", model.RecruitmentStatusClosed)
	return result.RowsAffected, result.Error
}

// ListExpiringSoonRecruitments 查询 withinDays 天内即将过期的招募（status=open，expires_at 不为 nil）
func (r *teamRepository) ListExpiringSoonRecruitments(ctx context.Context, withinDays int) ([]*model.Recruitment, error) {
	deadline := time.Now().Add(time.Duration(withinDays) * 24 * time.Hour)
	var recs []*model.Recruitment
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at BETWEEN ? AND ?",
			model.RecruitmentStatusOpen, time.Now(), deadline).
		Order("expires_at ASC").
		Find(&recs).Error
	return recs, err
}

// ==================== RecruitmentApplication 申请操作实现 ====================

// CreateApplication 创建申请记录
func (r *teamRepository) CreateApplication(ctx context.Context, app *model.RecruitmentApplication) error {
	if err := r.db.WithContext(ctx).Create(app).Error; err != nil {
		return err
	}
	return nil
}

// GetApplicationByID 根据 ID 查找申请
func (r *teamRepository) GetApplicationByID(ctx context.Context, id uint64) (*model.RecruitmentApplication, error) {
	var app model.RecruitmentApplication
	if err := r.db.WithContext(ctx).First(&app, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeApplicationNotFound)
		}
		return nil, err
	}
	return &app, nil
}

// GetApplicationsByRecruitmentID 查询招募下的申请列表（分页）
func (r *teamRepository) GetApplicationsByRecruitmentID(ctx context.Context, recruitmentID uint64, page, pageSize int) ([]*model.RecruitmentApplication, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.RecruitmentApplication{}).
		Where("recruitment_id = ?", recruitmentID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*model.RecruitmentApplication
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetApplicationsByApplicantID 查询用户提交的申请列表（分页）
func (r *teamRepository) GetApplicationsByApplicantID(ctx context.Context, applicantID uint64, page, pageSize int) ([]*model.RecruitmentApplication, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.RecruitmentApplication{}).
		Where("applicant_id = ?", applicantID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*model.RecruitmentApplication
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// UpdateApplicationStatus 更新申请状态
func (r *teamRepository) UpdateApplicationStatus(ctx context.Context, id uint64, status model.ApplicationStatus) error {
	result := r.db.WithContext(ctx).
		Model(&model.RecruitmentApplication{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// HasApplied 判断用户是否已申请某招募（排除已撤回的申请）
func (r *teamRepository) HasApplied(ctx context.Context, applicantID, recruitmentID uint64) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.RecruitmentApplication{}).
		Where("applicant_id = ? AND recruitment_id = ? AND status != ?",
			applicantID, recruitmentID, model.ApplicationStatusWithdrawn).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// GetApplicationByApplicantAndRecruitment 根据申请人和招募ID查找申请
func (r *teamRepository) GetApplicationByApplicantAndRecruitment(ctx context.Context, applicantID, recruitmentID uint64) (*model.RecruitmentApplication, error) {
	var app model.RecruitmentApplication
	if err := r.db.WithContext(ctx).
		Where("applicant_id = ? AND recruitment_id = ?", applicantID, recruitmentID).
		Order("created_at DESC").
		First(&app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeApplicationNotFound)
		}
		return nil, err
	}
	return &app, nil
}

// ==================== CollaborationReview 合作评价操作实现 ====================

// CreateReview 创建合作评价记录
func (r *teamRepository) CreateReview(ctx context.Context, review *model.CollaborationReview) error {
	if err := r.db.WithContext(ctx).Create(review).Error; err != nil {
		return err
	}
	return nil
}

// GetReviewByID 根据 ID 查找评价
func (r *teamRepository) GetReviewByID(ctx context.Context, id uint64) (*model.CollaborationReview, error) {
	var review model.CollaborationReview
	if err := r.db.WithContext(ctx).First(&review, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeReviewNotFound)
		}
		return nil, err
	}
	return &review, nil
}

// GetReviewsByRevieweeID 查询用户收到的评价列表（分页）
func (r *teamRepository) GetReviewsByRevieweeID(ctx context.Context, revieweeID uint64, page, pageSize int) ([]*model.CollaborationReview, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.CollaborationReview{}).
		Where("reviewee_id = ?", revieweeID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*model.CollaborationReview
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetReviewByProjectAndUsers 根据项目和用户对查找评价
func (r *teamRepository) GetReviewByProjectAndUsers(ctx context.Context, projectID, reviewerID, revieweeID uint64) (*model.CollaborationReview, error) {
	var review model.CollaborationReview
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND reviewer_id = ? AND reviewee_id = ?", projectID, reviewerID, revieweeID).
		First(&review).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeReviewNotFound)
		}
		return nil, err
	}
	return &review, nil
}

// HasReviewed 判断评价人在该项目中是否已对被评人发表过评价
func (r *teamRepository) HasReviewed(ctx context.Context, projectID, reviewerID, revieweeID uint64) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.CollaborationReview{}).
		Where("project_id = ? AND reviewer_id = ? AND reviewee_id = ?", projectID, reviewerID, revieweeID).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// AddSupplement 补充评价说明
// 已有补充说明则返回错误（只允许补充一次）
func (r *teamRepository) AddSupplement(ctx context.Context, reviewID uint64, supplement string) error {
	// 先查询是否已有补充说明
	var review model.CollaborationReview
	if err := r.db.WithContext(ctx).Select("id, supplement").First(&review, reviewID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.CodeError(apperrors.CodeReviewNotFound)
		}
		return err
	}
	if review.HasSupplement() {
		return apperrors.CodeError(apperrors.CodeReviewSupplementExists)
	}

	// 更新补充说明
	if err := r.db.WithContext(ctx).
		Model(&model.CollaborationReview{}).
		Where("id = ?", reviewID).
		Update("supplement", supplement).Error; err != nil {
		return err
	}
	return nil
}

// GetAverageRating 获取用户的平均评分和评价总数
func (r *teamRepository) GetAverageRating(ctx context.Context, revieweeID uint64) (float64, int64, error) {
	type result struct {
		AvgRating float64
		Count     int64
	}
	var res result
	if err := r.db.WithContext(ctx).
		Model(&model.CollaborationReview{}).
		Where("reviewee_id = ?", revieweeID).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as count").
		Scan(&res).Error; err != nil {
		return 0, 0, err
	}
	return res.AvgRating, res.Count, nil
}

// ==================== Notification 通知操作实现 ====================

// CreateNotification 创建通知记录
func (r *teamRepository) CreateNotification(ctx context.Context, n *model.Notification) error {
	if err := r.db.WithContext(ctx).Create(n).Error; err != nil {
		return err
	}
	return nil
}

// CreateNotifications 批量创建通知记录（单次 INSERT，100 条一批）
func (r *teamRepository) CreateNotifications(ctx context.Context, ns []*model.Notification) error {
	if len(ns) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(ns, 100).Error
}

// GetNotificationByID 根据 ID 查找通知
func (r *teamRepository) GetNotificationByID(ctx context.Context, id uint64) (*model.Notification, error) {
	var n model.Notification
	if err := r.db.WithContext(ctx).First(&n, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.CodeError(apperrors.CodeNotificationNotFound)
		}
		return nil, err
	}
	return &n, nil
}

// MarkAsRead 将单条通知标记为已读（只能标记属于该用户的通知）
func (r *teamRepository) MarkAsRead(ctx context.Context, id, userID uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeNotificationNotFound)
	}
	return nil
}

// MarkAllAsRead 将用户所有未读通知标记为已读
func (r *teamRepository) MarkAllAsRead(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}

// GetNotifications 分页查询用户通知列表（按创建时间倒序）
func (r *teamRepository) GetNotifications(ctx context.Context, params *ListNotificationsParams) ([]*model.Notification, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ?", params.UserID)

	// 按已读状态筛选
	if params.IsRead != nil {
		db = db.Where("is_read = ?", *params.IsRead)
	}

	// 多类型筛选（IN）优先于单类型筛选
	if len(params.Types) > 0 {
		db = db.Where("type IN ?", params.Types)
	} else if params.Type != "" {
		db = db.Where("type = ?", params.Type)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*model.Notification
	offset := (params.Page - 1) * params.PageSize
	if err := db.Order("created_at DESC").
		Offset(offset).
		Limit(params.PageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// CountUnread 统计用户未读通知数量
func (r *teamRepository) CountUnread(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// CountUnreadByTypes 按类型分组统计未读通知（单次 GROUP BY 查询，避免拉全表内存统计）
func (r *teamRepository) CountUnreadByTypes(ctx context.Context, userID uint64) (map[model.NotificationType]int64, error) {
	type row struct {
		Type  model.NotificationType
		Count int64
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Select("type, COUNT(*) AS count").
		Where("user_id = ? AND is_read = false", userID).
		Group("type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[model.NotificationType]int64, len(rows))
	for _, r := range rows {
		result[r.Type] = r.Count
	}
	return result, nil
}

// DeleteNotification 删除单条通知（需验证归属用户）
func (r *teamRepository) DeleteNotification(ctx context.Context, userID, notificationID uint64) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Delete(&model.Notification{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.CodeError(apperrors.CodeNotFound)
	}
	return nil
}

// DeleteReadNotifications 删除用户所有已读通知（硬删除）
func (r *teamRepository) DeleteReadNotifications(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND is_read = ?", userID, true).
		Delete(&model.Notification{}).Error
}

// ClearNotifications 清空用户所有通知（硬删除）
func (r *teamRepository) ClearNotifications(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.Notification{}).Error
}

// GetUserRecentEndorsements 查询用户收到的合作评价（reviewer != 自己），按时间倒序
func (r *teamRepository) GetUserRecentEndorsements(ctx context.Context, userID uint64, limit int) ([]*model.CollaborationReview, error) {
	var list []*model.CollaborationReview
	err := r.db.WithContext(ctx).
		Where("reviewee_id = ? AND reviewer_id != ?", userID, userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}
