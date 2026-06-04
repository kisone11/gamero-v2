// Package service 提供管理后台的业务逻辑层。
// 本文件包含以下管理功能的业务逻辑:
//   - 用户管理(列表/详情/封禁/认证级别/删除)
//   - 内容审核(举报处理/强制删帖/强制删评论)
//   - 话题管理(增删改/图标上传)
//   - 数据统计(概览/时序)
package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/storage"
	"go.uber.org/zap"
)


// ===========================
// 请求/响应数据结构
// ===========================

// AdminUserDetail 用户详情
type AdminUserDetail struct {
	User   *model.User   `json:"user"` // 用户基本信息
}

// AdminUserListItem 用户列表项
type AdminUserListItem struct {
	*model.User
}

// AdminReportItem 举报列表项(含目标摘要)
type AdminReportItem struct {
	*model.Report
	TargetTitle string `json:"target_title,omitempty"` // 举报目标摘要(帖子标题或评论内容片段)
}

// AdminCreateTopicReq 创建话题请求
type AdminCreateTopicReq struct {
	Name        string `json:"name"`        // 话题名称,唯一,最多64字符
	Description string `json:"description"` // 话题描述,最多256字符
}

// AdminUpdateTopicReq 更新话题请求
type AdminUpdateTopicReq struct {
	Name        *string `json:"name"`        // 话题名称(nil 表示不更新)
	Description *string `json:"description"` // 话题描述(nil 表示不更新)
}

// ===========================
// 接口定义
// ===========================

// AdminService 管理后台业务逻辑接口
type AdminService interface {
	// ===== 用户管理 =====

	// ListUsers 分页查询用户列表,支持关键词搜索
	ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]*AdminUserListItem, int64, error)
	// BatchBanUsers 批量封禁用户
	BatchBanUsers(ctx context.Context, adminID uint64, userIDs []uint64, durationHours int) error
	// GetUser 获取用户详情(含钱包信息)
	GetUser(ctx context.Context, id uint64) (*AdminUserDetail, error)
	// BanUser 封禁或解封用户;durationHours=0 表示永久封号;banned=false 时 durationHours 忽略
	BanUser(ctx context.Context, adminID, userID uint64, banned bool, durationHours int) error
	// AdminDeleteUser 软删除用户(不可删除 admin 账号或删除自己)
	AdminDeleteUser(ctx context.Context, adminID, userID uint64) error

	// ===== 内容审核 =====

	// ListReports 分页查询举报列表(含目标摘要)
	ListReports(ctx context.Context, targetType, status string, page, pageSize int) ([]*AdminReportItem, int64, error)
	// HandleReport 处理举报(标记已处理/已驳回,填写处理备注)
	// deleteContent:仅 newStatus="handled" 时有效,true 表示同步删除被举报内容
	HandleReport(ctx context.Context, adminID, reportID uint64, newStatus, note string, deleteContent bool) error
	// AdminDeletePost 强制删除帖子(不检查作者)
	AdminDeletePost(ctx context.Context, adminID, postID uint64) error
	// AdminDeleteComment 强制删除评论
	// commentType: "post" 帖子评论,"log" 日志评论
	AdminDeleteComment(ctx context.Context, adminID, commentID uint64, commentType string) error
	// AdminDeleteLog 强制删除日志(不检查作者)
	AdminDeleteLog(ctx context.Context, adminID, logID uint64) error
	// AdminCloseRecruitment 管理员强制关闭招募
	AdminCloseRecruitment(ctx context.Context, adminID, recruitmentID uint64) error
	// AdminDeleteProject 管理员软删除项目
	AdminDeleteProject(ctx context.Context, adminID, projectID uint64) error
	// ListProjects 分页查询项目
	ListProjects(ctx context.Context, keyword string, page, pageSize int) ([]*model.Project, int64, error)
	// ListPosts 分页查询帖子
	ListPosts(ctx context.Context, keyword string, page, pageSize int) ([]*model.Post, int64, error)
	// ListLogs 分页查询开发日志
	ListLogs(ctx context.Context, keyword string, page, pageSize int) ([]*model.DevLog, int64, error)

	// ===== 话题管理 =====

	// CreateTopic 创建话题
	CreateTopic(ctx context.Context, adminID uint64, req *AdminCreateTopicReq) (*model.Topic, error)
	// UpdateTopic 更新话题信息
	UpdateTopic(ctx context.Context, adminID, topicID uint64, req *AdminUpdateTopicReq) (*model.Topic, error)
	// DeleteTopic 软删除话题
	DeleteTopic(ctx context.Context, adminID, topicID uint64) error
	// SaveTopicIcon 保存话题图标 key(前端已直传后提交)
	SaveTopicIcon(ctx context.Context, adminID, topicID uint64, key string) error
	// ListTopics 查询话题
	ListTopics(ctx context.Context) ([]*model.Topic, error)

	// ===== 数据统计 =====

	// GetPlatformStats 获取平台概览统计数据
	GetPlatformStats(ctx context.Context) (*repository.PlatformStats, error)
	// GetDailyStats 获取过去 N 天每天的数据
	GetDailyStats(ctx context.Context, days int) ([]repository.DailyStats, error)

	// ===== 审计日志 =====

	// GetAuditLogs 分页查询管理员操作审计日志,adminID=0 表示查所有管理员
	GetAuditLogs(ctx context.Context, adminID uint64, page, pageSize int) ([]*model.AdminAuditLog, int64, error)

	// ExportUsers 导出用户列表为 CSV 字节,最多 10000 条
	ExportUsers(ctx context.Context, keyword string) ([]byte, error)

	// ===== 敏感词管理 =====

	// GetSensitiveWords 返回当前全量敏感词列表
	GetSensitiveWords() []string
	// AddSensitiveWords 动态添加敏感词(已有的自动跳过,内存+DB 双写)
	AddSensitiveWords(ctx context.Context, adminID uint64, words []string)
	// RemoveSensitiveWord 删除一个敏感词(内存+DB 同步删)
	RemoveSensitiveWord(ctx context.Context, word string)

	// ===== 用户解封 =====

	// UnbanUser 解封被封禁用户(清除封禁标记和到期时间)
	UnbanUser(ctx context.Context, adminID, userID uint64) error

	// ===== 项目管理 =====

	// BanProject 下架项目(设置 is_banned=true,通知项目 owner)
	BanProject(ctx context.Context, adminID, projectID uint64, reason string) error
	// UnbanProject 恢复项目上架(设置 is_banned=false,通知项目 owner)
	UnbanProject(ctx context.Context, adminID, projectID uint64) error
	// SetNotificationService 注入统一通知服务(由 router 在初始化时调用)。
	SetNotificationService(svc NotificationService)
}

// ===========================
// 实现
// ===========================

// adminService AdminService 的实现
type adminService struct {
	adminRepo     repository.AdminRepository
	teamRepo      repository.TeamRepository // 用于写通知(CreateNotification 在 TeamRepository 中)
	*NotificationClient
	devLogRepo    repository.DevLogRepository // 开发日志仓库
	stor          *storage.Client
	swRepo        repository.SensitiveWordRepository // 敏感词持久化仓库
}

// NewAdminService 创建 AdminService 实例
func NewAdminService(
	adminRepo repository.AdminRepository,
	teamRepo repository.TeamRepository,
	stor *storage.Client,
) AdminService {
	svc := &adminService{
		adminRepo: adminRepo,
		teamRepo:  teamRepo,
		stor:      stor,
		NotificationClient: &NotificationClient{},
	}
	return svc
}

// NewAdminServiceFull 创建完整功能 AdminService(含敏感词持久化 + 开发日志)
func NewAdminServiceFull(
	adminRepo repository.AdminRepository,
	teamRepo repository.TeamRepository,
	devLogRepo repository.DevLogRepository,
	swRepo repository.SensitiveWordRepository,
	stor *storage.Client,
) AdminService {
	return &adminService{
		adminRepo:  adminRepo,
		teamRepo:   teamRepo,
		devLogRepo: devLogRepo,
		swRepo:     swRepo,
		NotificationClient: &NotificationClient{},
		stor:       stor,
	}
}

// ===== 用户管理实现 =====

// ListUsers 分页查询用户列表

func (s *adminService) ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]*AdminUserListItem, int64, error) {
	offset := (page - 1) * pageSize
	users, total, err := s.adminRepo.ListUsers(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapMsg(apperrors.CodeInternalError, "查询用户列表失败", err)
	}

	items := make([]*AdminUserListItem, len(users))
	for i, u := range users {
		items[i] = &AdminUserListItem{User: u}
	}
	return items, total, nil
}

// BatchBanUsers 批量封禁用户
func (s *adminService) BatchBanUsers(ctx context.Context, adminID uint64, userIDs []uint64, durationHours int) error {
	if len(userIDs) == 0 {
		return apperrors.New(apperrors.CodeParamMissing, "用户 ID 不能为空")
	}
	if len(userIDs) > 100 {
		return apperrors.New(apperrors.CodeParamTooLong, "单次最多封禁100个用户")
	}
	for _, userID := range userIDs {
		if userID == adminID {
			return apperrors.New(apperrors.CodePermissionDenied, "不能封禁自己的账号")
		}
		if err := s.BanUser(ctx, adminID, userID, true, durationHours); err != nil {
			return err
		}
	}
	return nil
}

// GetUser 获取用户详情
func (s *adminService) GetUser(ctx context.Context, id uint64) (*AdminUserDetail, error) {
	user, err := s.adminRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "用户不存在")
	}

	detail := &AdminUserDetail{User: user}
	return detail, nil
}

// BanUser 封禁或解封用户;durationHours=0 表示永久封号;banned=false 时解封
func (s *adminService) BanUser(ctx context.Context, adminID, userID uint64, banned bool, durationHours int) error {
	// 确认用户存在
	user, err := s.adminRepo.GetUserByID(ctx, userID)
	if err != nil {
		return apperrors.New(apperrors.CodeNotFound, "用户不存在")
	}
	// 不允许封禁管理员账号
	if string(user.Role) == "admin" || string(user.Role) == "superadmin" {
		return apperrors.New(apperrors.CodePermissionDenied, "不允许封禁管理员账号")
	}

	var bannedUntil *time.Time
	if banned && durationHours > 0 {
		t := time.Now().Add(time.Duration(durationHours) * time.Hour)
		bannedUntil = &t
	}

	if err := s.adminRepo.BanUser(ctx, userID, banned, bannedUntil); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "操作失败", err)
	}

	// 写入审计日志(失败不中断主流程)
	action := "ban_user"
	if !banned {
		action = "unban_user"
	}
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: "user",
		TargetID:   userID,
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}

	return nil
}

// AdminDeleteUser 软删除用户
// 约束:不可删除自己,不可删除 admin 账号
func (s *adminService) AdminDeleteUser(ctx context.Context, adminID, userID uint64) error {
	if adminID == userID {
		return apperrors.New(apperrors.CodePermissionDenied, "不能删除自己的账号")
	}

	user, err := s.adminRepo.GetUserByID(ctx, userID)
	if err != nil {
		return apperrors.New(apperrors.CodeNotFound, "用户不存在")
	}

	if string(user.Role) == "admin" {
		return apperrors.New(apperrors.CodePermissionDenied, "不允许删除管理员账号")
	}

	if err := s.adminRepo.AdminDeleteUser(ctx, userID); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "删除用户失败", err)
	}
	return nil
}

// ===== 内容审核实现 =====

// ListReports 分页查询举报列表(含目标摘要)
func (s *adminService) ListReports(ctx context.Context, targetType, status string, page, pageSize int) ([]*AdminReportItem, int64, error) {
	offset := (page - 1) * pageSize
	reports, total, err := s.adminRepo.ListReports(ctx, targetType, status, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapMsg(apperrors.CodeInternalError, "查询举报列表失败", err)
	}

	items := make([]*AdminReportItem, len(reports))
	for i, r := range reports {
		items[i] = &AdminReportItem{Report: r}
	}
	return items, total, nil
}

// HandleReport 处理举报
// newStatus 合法值:handled(已处理)/ rejected(已驳回)
// deleteContent:newStatus="handled" 时可选同步删除被举报内容
func (s *adminService) HandleReport(ctx context.Context, adminID, reportID uint64, newStatus, note string, deleteContent bool) error {
	// 校验状态值
	if newStatus != "handled" && newStatus != "rejected" {
		return apperrors.New(apperrors.CodeParamInvalid, "无效的处理状态,允许值:handled / rejected")
	}

	// 查询举报记录(用于后续通知和联动删除)
	report, err := s.adminRepo.GetReportByID(ctx, reportID)
	if err != nil {
		return err
	}
	if report.Status != "pending" && report.Status != "escalated" {
		return apperrors.New(apperrors.CodeConflict, "该举报已处理,无法重复操作")
	}

	if err := s.adminRepo.HandleReport(ctx, reportID, newStatus, note); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "处理举报失败", err)
	}

	// handled + deleteContent:联动删除被举报内容
	if newStatus == "handled" && deleteContent {
		switch report.TargetType {
		case "post":
			if err := s.adminRepo.AdminDeletePost(ctx, report.TargetID); err != nil {
				logger.Warn("failed to delete reported post", zap.Uint64("post_id", report.TargetID), zap.Error(err))
			}
		case "comment", "log_comment":
			commentType := "post"
			if report.TargetType == "log_comment" {
				commentType = "log"
			}
			if err := s.adminRepo.AdminDeleteComment(ctx, report.TargetID, commentType); err != nil {
				logger.Warn("failed to delete reported comment", zap.Uint64("comment_id", report.TargetID), zap.Error(err))
			}
		}
	}

	// 通知举报人处理结果(失败不阻断)
	var notifTitle, notifContent string
	if newStatus == "handled" {
		notifTitle = "你的举报已处理"
		notifContent = "你的举报已被核实,我们已对相关内容采取处置措施。"
		if note != "" {
			notifContent += "备注:" + note
		}
	} else {
		notifTitle = "你的举报未通过审核"
		notifContent = "经审核,你的举报内容不符合规定,已被驳回。"
		if note != "" {
			notifContent += "备注:" + note
		}
	}
	s.Send(ctx, &SendNotificationReq{
		UserID: report.ReporterID,
		Type: model.NotificationTypeReportHandled,
		Title: notifTitle,
		Content: notifContent,
		Metadata: map[string]interface{}{"report_id": reportID, "target_type": report.TargetType, "target_id": report.TargetID},
	})

	// 写入审计日志
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "handle_report",
		TargetType: "report",
		TargetID:   reportID,
		Note:       note,
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}

	return nil
}

// AdminDeletePost 强制删除帖子
func (s *adminService) AdminDeletePost(ctx context.Context, adminID, postID uint64) error {
	if err := s.adminRepo.AdminDeletePost(ctx, postID); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "删除帖子失败", err)
	}

	// 写入审计日志
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "delete_post",
		TargetType: "post",
		TargetID:   postID,
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}

	return nil
}

// AdminDeleteComment 强制删除评论
func (s *adminService) AdminDeleteComment(ctx context.Context, adminID, commentID uint64, commentType string) error {
	if commentType != "post" && commentType != "log" {
		return apperrors.New(apperrors.CodeParamInvalid, "无效的评论类型,允许值:post / log")
	}
	if err := s.adminRepo.AdminDeleteComment(ctx, commentID, commentType); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "删除评论失败", err)
	}
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "delete_comment",
		TargetType: commentType + "_comment",
		TargetID:   commentID,
		Note:     fmt.Sprintf("删除 %s 评论 %d", commentType, commentID),
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}
	return nil
}

// AdminDeleteLog 强制删除日志
func (s *adminService) AdminDeleteLog(ctx context.Context, adminID, logID uint64) error {
	// 确认日志存在
	if _, err := s.devLogRepo.GetLogByID(ctx, logID); err != nil {
		return apperrors.New(apperrors.CodeNotFound, "日志不存在")
	}

	// 软删除日志
	if err := s.devLogRepo.DeleteLog(ctx, logID); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "删除日志失败", err)
	}

	// 写入审计日志
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "delete_log",
		TargetType: "log",
		TargetID:   logID,
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}

	return nil
}

// AdminCloseRecruitment 管理员强制关闭招募
func (s *adminService) AdminCloseRecruitment(ctx context.Context, adminID, recruitmentID uint64) error {
	if s.teamRepo == nil {
		return apperrors.New(apperrors.CodeInternalError, "招募仓储未初始化")
	}
	if _, err := s.teamRepo.GetRecruitmentByID(ctx, recruitmentID); err != nil {
		return err
	}
	if err := s.teamRepo.CloseRecruitment(ctx, recruitmentID); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "关闭招募失败", err)
	}
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "close_recruitment",
		TargetType: "recruitment",
		TargetID:   recruitmentID,
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}
	return nil
}

// AdminDeleteProject 管理员软删除项目
func (s *adminService) AdminDeleteProject(ctx context.Context, adminID, projectID uint64) error {
	if _, err := s.adminRepo.GetProjectByIDForAdmin(ctx, projectID); err != nil {
		return err
	}
	if err := s.adminRepo.DB().Model(&model.Project{}).Where("id = ?", projectID).Update("deleted_at", time.Now()).Error; err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "删除项目失败", err)
	}
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "delete_project",
		TargetType: "project",
		TargetID:   projectID,
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}
	return nil
}

// ListProjects 分页查询项目
func (s *adminService) ListProjects(ctx context.Context, keyword string, page, pageSize int) ([]*model.Project, int64, error) {
	offset := (page - 1) * pageSize
	projects, total, err := s.adminRepo.ListProjects(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapMsg(apperrors.CodeInternalError, "查询项目列表失败", err)
	}
	return projects, total, nil
}

// ListPosts 分页查询帖子
func (s *adminService) ListPosts(ctx context.Context, keyword string, page, pageSize int) ([]*model.Post, int64, error) {
	offset := (page - 1) * pageSize
	posts, total, err := s.adminRepo.ListPosts(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapMsg(apperrors.CodeInternalError, "查询帖子列表失败", err)
	}
	return posts, total, nil
}

// ListLogs 分页查询开发日志
func (s *adminService) ListLogs(ctx context.Context, keyword string, page, pageSize int) ([]*model.DevLog, int64, error) {
	offset := (page - 1) * pageSize
	logs, total, err := s.adminRepo.ListLogs(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapMsg(apperrors.CodeInternalError, "查询开发日志列表失败", err)
	}
	return logs, total, nil
}

// ===== 话题管理实现 =====

// CreateTopic 创建话题
func (s *adminService) CreateTopic(ctx context.Context, adminID uint64, req *AdminCreateTopicReq) (*model.Topic, error) {
	if req.Name == "" {
		return nil, apperrors.New(apperrors.CodeParamMissing, "话题名称不能为空")
	}
	if len([]rune(req.Name)) > 64 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "话题名称不能超过64字符")
	}
	if len([]rune(req.Description)) > 256 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "话题描述不能超过256字符")
	}

	topic := &model.Topic{
		Name:        req.Name,
		Description: req.Description,
		IsDefault:   false,
	}

	if err := s.adminRepo.AdminCreateTopic(ctx, topic); err != nil {
		return nil, apperrors.WrapMsg(apperrors.CodeInternalError, "创建话题失败", err)
	}
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "create_topic",
		TargetType: "topic",
		TargetID:   topic.ID,
		Note:     fmt.Sprintf("创建话题「%s」", topic.Name),
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}
	return topic, nil
}

// UpdateTopic 更新话题信息
func (s *adminService) UpdateTopic(ctx context.Context, adminID, topicID uint64, req *AdminUpdateTopicReq) (*model.Topic, error) {
	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			return nil, apperrors.New(apperrors.CodeParamMissing, "话题名称不能为空")
		}
		if len([]rune(*req.Name)) > 64 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "话题名称不能超过64字符")
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		if len([]rune(*req.Description)) > 256 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "话题描述不能超过256字符")
		}
		updates["description"] = *req.Description
	}

	if len(updates) == 0 {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "没有需要更新的字段")
	}

	if err := s.adminRepo.AdminUpdateTopic(ctx, topicID, updates); err != nil {
		return nil, apperrors.WrapMsg(apperrors.CodeInternalError, "更新话题失败", err)
	}

	// 查询并返回更新后的话题
	topic, err := s.adminRepo.GetTopicByIDForAdmin(ctx, topicID)
	if err != nil {
		return nil, apperrors.New(apperrors.CodeNotFound, "话题不存在")
	}
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "update_topic",
		TargetType: "topic",
		TargetID:   topicID,
		Note:     fmt.Sprintf("更新话题 %d", topicID),
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}
	return topic, nil
}

// DeleteTopic 软删除话题
func (s *adminService) DeleteTopic(ctx context.Context, adminID, topicID uint64) error {
	// 确认话题存在
	_, err := s.adminRepo.GetTopicByIDForAdmin(ctx, topicID)
	if err != nil {
		return apperrors.New(apperrors.CodeNotFound, "话题不存在")
	}

	if err := s.adminRepo.AdminDeleteTopic(ctx, topicID); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "删除话题失败", err)
	}
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "delete_topic",
		TargetType: "topic",
		TargetID:   topicID,
		Note:     fmt.Sprintf("删除话题 %d", topicID),
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}
	return nil
}

// SaveTopicIcon 保存话题图标 key(前端已通过预签名 URL 直传后提交)
func (s *adminService) SaveTopicIcon(ctx context.Context, adminID, topicID uint64, key string) error {
	if key == "" {
		return apperrors.New(apperrors.CodeParamInvalid, "key 不能为空")
	}
	// 确认话题存在
	_, err := s.adminRepo.GetTopicByIDForAdmin(ctx, topicID)
	if err != nil {
		return apperrors.New(apperrors.CodeNotFound, "话题不存在")
	}
	// 更新话题图标 key
	if err := s.adminRepo.AdminUpdateTopic(ctx, topicID, map[string]interface{}{
		"icon_key": key,
	}); err != nil {
		return apperrors.WrapMsg(apperrors.CodeInternalError, "更新话题图标失败", err)
	}
	return nil
}

// ListTopics 查询话题
func (s *adminService) ListTopics(ctx context.Context) ([]*model.Topic, error) {
	topics, err := s.adminRepo.ListTopics(ctx)
	if err != nil {
		return nil, apperrors.WrapMsg(apperrors.CodeInternalError, "查询话题列表失败", err)
	}
	return topics, nil
}


// ===== 数据统计实现 =====

// GetPlatformStats 获取平台概览统计数据
func (s *adminService) GetPlatformStats(ctx context.Context) (*repository.PlatformStats, error) {
	stats, err := s.adminRepo.GetPlatformStats(ctx)
	if err != nil {
		return nil, apperrors.WrapMsg(apperrors.CodeInternalError, "获取平台统计数据失败", err)
	}
	return stats, nil
}

// GetDailyStats 获取过去 N 天每天的数据
func (s *adminService) GetDailyStats(ctx context.Context, days int) ([]repository.DailyStats, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	result, err := s.adminRepo.GetDailyStats(ctx, days)
	if err != nil {
		return nil, apperrors.WrapMsg(apperrors.CodeInternalError, "获取每日统计数据失败", err)
	}
	return result, nil
}

// GetAuditLogs 分页查询管理员操作审计日志(已在下方 ===== 审计日志实现 ===== 区域实现)

// ===== 审计日志实现 =====

// GetAuditLogs 分页查询管理员操作审计日志
// adminID=0 表示查询所有管理员的日志
func (s *adminService) GetAuditLogs(ctx context.Context, adminID uint64, page, pageSize int) ([]*model.AdminAuditLog, int64, error) {
	offset := (page - 1) * pageSize
	logs, total, err := s.adminRepo.ListAuditLogs(ctx, adminID, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapMsg(apperrors.CodeInternalError, "查询审计日志失败", err)
	}
	return logs, total, nil
}

// ExportUsers 导出用户列表为 CSV,最多 10000 条。
func (s *adminService) ExportUsers(ctx context.Context, keyword string) ([]byte, error) {
	users, _, err := s.ListUsers(ctx, keyword, 1, 10000)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	// 写表头
	if err := w.Write([]string{"ID", "Username", "Email", "Phone", "Role", "CreatedAt", "BannedUntil"}); err != nil {
		logger.Warn("failed to write CSV header", zap.Error(err))
	}
	for _, item := range users {
		if item.User == nil {
			continue
		}
		bannedUntil := ""
		if item.BannedUntil != nil {
			bannedUntil = item.BannedUntil.Format("2006-01-02 15:04:05")
		}
		if err := w.Write([]string{
			strconv.FormatUint(item.ID, 10),
			item.Username,
			item.Email,
			item.Phone,
			string(item.Role),
			item.CreatedAt.Format("2006-01-02 15:04:05"),
			bannedUntil,
		}); err != nil {
			logger.Warn("failed to write CSV row", zap.Error(err))
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// UnbanUser 解封被封禁用户
func (s *adminService) UnbanUser(ctx context.Context, adminID, userID uint64) error {
	if _, err := s.adminRepo.GetUserByID(ctx, userID); err != nil {
		return apperrors.New(apperrors.CodeNotFound, "用户不存在")
	}
	return s.BanUser(ctx, adminID, userID, false, 0)
}

// BanProject 下架项目(设置 is_banned=true,发通知给项目 owner)
func (s *adminService) BanProject(ctx context.Context, adminID, projectID uint64, reason string) error {
	proj, err := s.adminRepo.GetProjectByIDForAdmin(ctx, projectID)
	if err != nil {
		return err
	}
	if proj.IsBanned {
		return apperrors.New(apperrors.CodeParamInvalid, "项目已处于下架状态")
	}
	if err := s.adminRepo.BanProject(ctx, projectID, true, reason); err != nil {
		return err
	}
	// 写审计日志
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "ban_project",
		TargetType: "project",
		TargetID:   projectID,
		Note:       reason,
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}
	// 通知项目 owner
	if s.teamRepo != nil {
		msg := "您的项目「" + proj.Name + "」已被管理员下架"
		if reason != "" {
			msg += ",原因:" + reason
		}
		meta := fmt.Sprintf(`{"target_type":"project","target_id":%d}`, projectID)
		n := &model.Notification{
			UserID:   proj.OwnerID,
			Type:     model.NotificationTypeProjectBanned,
			Title:    "项目已下架",
			Content:  msg,
			Metadata: meta,
		}
		s.SendModel(ctx, n)
	}
	return nil
}

// UnbanProject 恢复项目上架(设置 is_banned=false,发通知给项目 owner)
func (s *adminService) UnbanProject(ctx context.Context, adminID, projectID uint64) error {
	proj, err := s.adminRepo.GetProjectByIDForAdmin(ctx, projectID)
	if err != nil {
		return err
	}
	if !proj.IsBanned {
		return apperrors.New(apperrors.CodeParamInvalid, "项目当前未处于下架状态")
	}
	if err := s.adminRepo.BanProject(ctx, projectID, false, ""); err != nil {
		return err
	}
	// 写审计日志
	if err := s.adminRepo.CreateAuditLog(ctx, &model.AdminAuditLog{
		AdminID:    adminID,
		Action:     "unban_project",
		TargetType: "project",
		TargetID:   projectID,
	}); err != nil {
		logger.Warn("failed to create audit log", zap.Error(err))
	}
	// 通知项目 owner
	if s.teamRepo != nil {
		meta := fmt.Sprintf(`{"target_type":"project","target_id":%d}`, projectID)
		n := &model.Notification{
			UserID:   proj.OwnerID,
			Type:     model.NotificationTypeProjectUnbanned,
			Title:    "项目已恢复上架",
			Content:  "您的项目「" + proj.Name + "」已由管理员恢复上架,用户可正常访问。",
			Metadata: meta,
		}
		s.SendModel(ctx, n)
	}
	return nil
}

// ===========================
// 敏感词管理
// ===========================

// GetSensitiveWords 返回当前内存词库
func (s *adminService) GetSensitiveWords() []string {
	return filter.Get().Words()
}

// AddSensitiveWords 内存 + DB 双写,已有词自动跳过
func (s *adminService) AddSensitiveWords(ctx context.Context, adminID uint64, words []string) {
	// 1. 内存即时生效
	filter.Get().AddWords(words)
	// 2. 异步持久化(不阻塞主调用路径)
	if s.swRepo != nil {
		go func() {
			if err := s.swRepo.Add(context.Background(), words, adminID); err != nil {
				logger.Get().Warn("敏感词 DB 写入失败", zap.Error(err))
			}
		}()
	}
}

// RemoveSensitiveWord 内存 + DB 双删
func (s *adminService) RemoveSensitiveWord(ctx context.Context, word string) {
	// 1. 内存即时生效
	filter.Get().RemoveWord(word)
	// 2. 异步持久化
	if s.swRepo != nil {
		go func() {
			if err := s.swRepo.Remove(context.Background(), word); err != nil {
				logger.Get().Warn("敏感词 DB 删除失败", zap.Error(err))
			}
		}()
	}
}
