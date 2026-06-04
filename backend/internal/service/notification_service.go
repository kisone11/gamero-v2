// Package service 通知业务逻辑层。
// notification_service.go：统一通知写入、查询、标记已读、未读计数等。
//
// 设计原则：
//  - 所有业务 service 写通知必须通过 NotificationService.Send，
//    禁止直接调用 teamRepo.CreateNotification。
//  - Send 同步写 DB，成功后异步自增缓存计数 + WS 实时推送。
//  - 读操作（列表/标记已读/未读计数）通过 cache-aside 策略减少 DB 压力。
package service

import (
	"go.uber.org/zap"
	"github.com/gamero/gamero/pkg/logger"
	"context"
	"encoding/json"
	"time"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/cache"
	apperrors "github.com/gamero/gamero/pkg/errors"
)

// ===========================
// 接口定义
// ===========================

// NotificationService 统一通知服务接口。
type NotificationService interface {
	// Send 写入一条通知，成功后异步 cache 自增 + WS 推送。
	// senderID=0 表示系统消息。
	Send(ctx context.Context, req *SendNotificationReq) error

	// SendModel 直接传入 *model.Notification 写入，成功后异步副作用。
	SendModel(ctx context.Context, n *model.Notification) error

	// SendBatch 批量写通知（同一类型发给多人），失败不阻断。
	SendBatch(ctx context.Context, reqs []*SendNotificationReq)

	// GetMyNotifications 分页查询用户通知列表。
	GetMyNotifications(ctx context.Context, userID uint64, page, pageSize int, onlyUnread bool) ([]*NotificationItem, int64, error)

	// MarkAsRead 将单条通知标记为已读，删除未读缓存。
	MarkAsRead(ctx context.Context, userID, notificationID uint64) error

	// MarkAllAsRead 将用户所有通知标记为已读，删除未读缓存。
	MarkAllAsRead(ctx context.Context, userID uint64) error

	// GetUnreadCount 获取用户未读通知数量（cache-aside，永不过期，手动失效）。
	GetUnreadCount(ctx context.Context, userID uint64) (int64, error)

	// Delete 删除单条通知（需验证归属）。
	Delete(ctx context.Context, userID, notificationID uint64) error

	// DeleteRead 删除用户所有已读通知。
	DeleteRead(ctx context.Context, userID uint64) error

	// Clear 清空用户所有通知。
	Clear(ctx context.Context, userID uint64) error

	// GetNotificationCategories 获取各分类未读通知数。
	GetNotificationCategories(ctx context.Context, userID uint64) ([]*NotificationCategoryCount, error)

	// GetMyNotificationsByType 按类型过滤通知列表。
	GetMyNotificationsByType(ctx context.Context, userID uint64, notifType string, page, pageSize int, onlyUnread bool) ([]*NotificationItem, int64, error)
}

// SendNotificationReq 写通知请求
type SendNotificationReq struct {
	UserID   uint64                 // 接收者（必填）
	SenderID uint64                 // 发送者，0=系统
	Type     model.NotificationType // 通知类型（必填）
	Title    string                 // 标题（必填）
	Content  string                 // 正文（必填）
	Metadata map[string]interface{} // 附加元数据（可为 nil）
}

// NotificationCategoryCount 通知分类未读数统计
type NotificationCategoryCount struct {
	TypeGroup   string `json:"type_group"`   // social / project / system / payment
	UnreadCount int64  `json:"unread_count"`
}

// notificationTypeGroups 各分类包含的通知类型
var notificationTypeGroups = map[string][]model.NotificationType{
	"social": {
		model.NotificationTypeMention,
		model.NotificationTypeNewFollower,
		model.NotificationTypePostLiked,
		model.NotificationTypePostCommented,
		model.NotificationTypeLogLiked,
		model.NotificationTypeLogCommented,
	},
	"project": {
		model.NotificationTypeRecruitmentApplied,
		model.NotificationTypeApplicationApproved,
		model.NotificationTypeApplicationRejected,
		model.NotificationTypeProjectStatusChanged,
		model.NotificationTypeTalentInvited,
		model.NotificationTypeInviteAccepted,
		model.NotificationTypeInviteDeclined,
		model.NotificationTypeProjectBanned,
		model.NotificationTypeProjectUnbanned,
	},
	"system": {
		model.NotificationTypeSystem,
		model.NotificationTypeReportHandled,
	},
}

// NotificationItem 通知列表 DTO（在 model.Notification 基础上增加解析后的 Metadata）
type NotificationItem struct {
	ID        uint64                 `json:"id"`
	SenderID  uint64                 `json:"sender_id"`
	Type      model.NotificationType `json:"type"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	IsRead    bool                   `json:"is_read"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// ===========================
// 实现
// ===========================

type notificationService struct {
	repo     repository.TeamRepository
	wsPusher NotificationPusher // 可为 nil，降级安全
}

// NewNotificationService 创建通知服务实例。
func NewNotificationService(repo repository.TeamRepository, wsPusher ...NotificationPusher) NotificationService {
	svc := &notificationService{repo: repo}
	if len(wsPusher) > 0 && wsPusher[0] != nil {
		svc.wsPusher = wsPusher[0]
	}
	return svc
}

// Send 同步写 DB，成功后异步 cache 自增 + WS 推送。
func (s *notificationService) Send(ctx context.Context, req *SendNotificationReq) error {
	meta := "{}"
	if len(req.Metadata) > 0 {
		b, err := json.Marshal(req.Metadata)
		if err == nil {
			meta = string(b)
		}
	}
	n := &model.Notification{
		UserID:   req.UserID,
		SenderID: req.SenderID,
		Type:     req.Type,
		Title:    req.Title,
		Content:  req.Content,
		Metadata: meta,
	}
	if err := s.repo.CreateNotification(ctx, n); err != nil {
		return err
	}
	// 异步：cache 自增 + WS 推送
	go s.afterSend(context.Background(), n)
	return nil
}

// SendBatch 批量写通知，失败忽略（适合关注者广播、运营公告等）。
func (s *notificationService) SendBatch(ctx context.Context, reqs []*SendNotificationReq) {
	if len(reqs) == 0 {
		return
	}
	// 构建通知模型列表
	ns := make([]*model.Notification, 0, len(reqs))
	for _, req := range reqs {
		metaStr := ""
		if req.Metadata != nil {
			if b, err := json.Marshal(req.Metadata); err == nil {
				metaStr = string(b)
			}
		}
		ns = append(ns, &model.Notification{
			UserID:   req.UserID,
			SenderID: req.SenderID,
			Type:     req.Type,
			Title:    req.Title,
			Content:  req.Content,
			Metadata: metaStr,
		})
	}
	// 单次批量 INSERT（100 条一批）替代之前的串行 N 次写入
	if err := s.repo.CreateNotifications(ctx, ns); err != nil {
		logger.Warn("failed to batch create notifications", zap.Int("count", len(ns)), zap.Error(err))
	}
	// 异步副作用（WS 推送、缓存更新）
	go func() {
		for _, n := range ns {
			s.afterSend(context.Background(), n)
		}
	}()
}

// SendModel 直接传入 *model.Notification 写入，成功后异步副作用。
func (s *notificationService) SendModel(ctx context.Context, n *model.Notification) error {
	if err := s.repo.CreateNotification(ctx, n); err != nil {
		return err
	}
	go s.afterSend(context.Background(), n)
	return nil
}

// afterSend 通知写入成功后的副作用：cache 自增 + WS 推送（异步调用，panic 安全）。
func (s *notificationService) afterSend(ctx context.Context, n *model.Notification) {
	defer func() { recover() }() //nolint:errcheck

	// 1. 未读计数缓存自增 + 获取最新值（避免重复查询 DB）
	cacheKey := cache.UserUnreadKey(n.UserID)
	var unreadCount int64 = -1
	if exists, _ := cache.Exists(ctx, cacheKey); exists {
		if count, err := cache.Incr(ctx, cacheKey); err == nil {
			unreadCount = count
		} else {
			logger.Warn("failed to increment cache", zap.Error(err))
		}
	}
	// 通知分类未读数缓存删除（新通知到来后失效，30s TTL 下次请求重建）
	if err := cache.Del(ctx, cache.UserNotifCategoriesKey(n.UserID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}

	// 2. WS 实时推送（可选）
	if s.wsPusher != nil {
		s.wsPusher.PushNotification(ctx, n.UserID, n)
		// 缓存未命中时回退 DB 查询
		if unreadCount < 0 {
			if count, err := s.repo.CountUnread(ctx, n.UserID); err == nil {
				unreadCount = count
			}
		}
		if unreadCount >= 0 {
			s.wsPusher.PushUnreadCount(n.UserID, unreadCount)
		}
	}
}

// GetMyNotifications 分页查询通知列表，返回 DTO 方便前端直接使用。
func (s *notificationService) GetMyNotifications(ctx context.Context, userID uint64, page, pageSize int, onlyUnread bool) ([]*NotificationItem, int64, error) {
	params := &repository.ListNotificationsParams{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	}
	if onlyUnread {
		f := false
		params.IsRead = &f
	}
	list, total, err := s.repo.GetNotifications(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*NotificationItem, 0, len(list))
	for _, n := range list {
		item := &NotificationItem{
			ID:        n.ID,
			SenderID:  n.SenderID,
			Type:      n.Type,
			Title:     n.Title,
			Content:   n.Content,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt,
		}
		if n.Metadata != "" && n.Metadata != "{}" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(n.Metadata), &meta); err == nil {
				item.Metadata = meta
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}

// MarkAsRead 标记单条已读，删除未读缓存（下次从 DB 重建）。
func (s *notificationService) MarkAsRead(ctx context.Context, userID, notificationID uint64) error {
	if err := s.repo.MarkAsRead(ctx, notificationID, userID); err != nil {
		return err
	}
	if err := cache.Del(ctx, cache.UserUnreadKey(userID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	if err := cache.Del(ctx, cache.UserNotifCategoriesKey(userID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	return nil
}

// MarkAllAsRead 标记全部已读，删除未读缓存。
func (s *notificationService) MarkAllAsRead(ctx context.Context, userID uint64) error {
	if err := s.repo.MarkAllAsRead(ctx, userID); err != nil {
		return err
	}
	if err := cache.Del(ctx, cache.UserUnreadKey(userID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	if err := cache.Del(ctx, cache.UserNotifCategoriesKey(userID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	return nil
}

// GetUnreadCount cache-aside：永不过期，手动失效策略。
func (s *notificationService) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	cacheKey := cache.UserUnreadKey(userID)
	var cnt int64
	if hit, _ := cache.GetJSON(ctx, cacheKey, &cnt); hit {
		return cnt, nil
	}
	count, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		return 0, err
	}
	if err := cache.SetJSON(ctx, cacheKey, count, cache.TTLUnreadCount); err != nil {
		logger.Warn("failed to set cache", zap.Error(err))
	}
	return count, nil
}

// Delete 删除单条通知（归属验证由 repo 层处理）。
func (s *notificationService) Delete(ctx context.Context, userID, notificationID uint64) error {
	n, err := s.repo.GetNotificationByID(ctx, notificationID)
	if err != nil {
		return err
	}
	if n == nil || n.UserID != userID {
		return apperrors.New(apperrors.CodeForbidden, "无权删除此通知")
	}
	if err := s.repo.DeleteNotification(ctx, userID, notificationID); err != nil {
		return err
	}
	// 若该条未读，删除未读缓存
	if !n.IsRead {
		if err := cache.Del(ctx, cache.UserUnreadKey(userID)); err != nil {
			logger.Warn("failed to delete cache", zap.Error(err))
		}
		if err := cache.Del(ctx, cache.UserNotifCategoriesKey(userID)); err != nil {
			logger.Warn("failed to delete cache", zap.Error(err))
		}
	}
	return nil
}

// Clear 清空用户所有通知，同步删除未读缓存。
func (s *notificationService) Clear(ctx context.Context, userID uint64) error {
	if err := s.repo.ClearNotifications(ctx, userID); err != nil {
		return err
	}
	if err := cache.Del(ctx, cache.UserUnreadKey(userID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	if err := cache.Del(ctx, cache.UserNotifCategoriesKey(userID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	return nil
}

// DeleteRead 删除用户所有已读通知，已读通知不影响未读计数。
func (s *notificationService) DeleteRead(ctx context.Context, userID uint64) error {
	if err := s.repo.DeleteReadNotifications(ctx, userID); err != nil {
		return err
	}
	// 已读通知删除后不影响未读计数，但应让分类缓存失效
	if err := cache.Del(ctx, cache.UserNotifCategoriesKey(userID)); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	return nil
}

// notifTypeGroups 通知分类映射
var notifTypeGroups = map[model.NotificationType]string{
	// social
	model.NotificationTypeMention:        "social",
	model.NotificationTypeNewFollower:    "social",
	model.NotificationTypePostLiked:      "social",
	model.NotificationTypePostCommented:  "social",
	model.NotificationTypeLogLiked:       "social",
	model.NotificationTypeLogCommented:   "social",
	// project
	model.NotificationTypeRecruitmentApplied:  "project",
	model.NotificationTypeApplicationApproved: "project",
	model.NotificationTypeApplicationRejected: "project",
	model.NotificationTypeProjectStatusChanged: "project",
	model.NotificationTypeTalentInvited:        "project",
	model.NotificationTypeInviteAccepted:       "project",
	model.NotificationTypeInviteDeclined:       "project",
	model.NotificationTypeProjectBanned:        "project",
	model.NotificationTypeProjectUnbanned:      "project",
	// system
	model.NotificationTypeSystem:             "system",
	model.NotificationTypeReportHandled:      "system",
}

// GetNotificationCategories 获取各分类未读通知数。
// 使用 GROUP BY 单次查询，避免拉全量未读通知内存统计。
func (s *notificationService) GetNotificationCategories(ctx context.Context, userID uint64) ([]*NotificationCategoryCount, error) {
	// cache-aside：短 TTL（30s）防止高频请求打穿 DB
	cacheKey := cache.UserNotifCategoriesKey(userID)
	var cached []*NotificationCategoryCount
	if hit, _ := cache.GetJSON(ctx, cacheKey, &cached); hit {
		return cached, nil
	}

	typeCounts, err := s.repo.CountUnreadByTypes(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 将类型映射到分组
	counts := map[string]int64{
		"social":  0,
		"project": 0,
		"system":  0,
	}
	for nType, cnt := range typeCounts {
		if group, ok := notifTypeGroups[nType]; ok {
			counts[group] += cnt
		} else {
			counts["system"] += cnt // 未分类的归入 system
		}
	}

	result := make([]*NotificationCategoryCount, 0, 3)
	for _, g := range []string{"social", "project", "system"} {
		result = append(result, &NotificationCategoryCount{
			TypeGroup:   g,
			UnreadCount: counts[g],
		})
	}

	// 写入缓存，失败不影响返回结果
	if err := cache.SetJSON(ctx, cacheKey, result, cache.TTLNotifCategories); err != nil {
		logger.Warn("failed to set cache", zap.Error(err))
	}
	return result, nil
}

// GetMyNotificationsByType 按类型分组过滤通知列表。
// GetMyNotificationsByType 按类型分组过滤通知列表。
// notifType 为分组名（social/project/system/payment）或单个通知类型，为空表示不过滤。
func (s *notificationService) GetMyNotificationsByType(ctx context.Context, userID uint64, notifType string, page, pageSize int, onlyUnread bool) ([]*NotificationItem, int64, error) {
	params := &repository.ListNotificationsParams{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	}
	if onlyUnread {
		f := false
		params.IsRead = &f
	}

	// 将类型过滤条件下沉至 DB 层，保证 total 准确
	if types, ok := notificationTypeGroups[notifType]; ok {
		// 分组名：IN 查询
		params.Types = types
	} else if notifType != "" {
		// 单个类型字符串
		params.Type = model.NotificationType(notifType)
	}

	list, total, err := s.repo.GetNotifications(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*NotificationItem, 0, len(list))
	for _, n := range list {
		item := &NotificationItem{
			ID:        n.ID,
			SenderID:  n.SenderID,
			Type:      n.Type,
			Title:     n.Title,
			Content:   n.Content,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt,
		}
		if n.Metadata != "" && n.Metadata != "{}" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(n.Metadata), &meta); err == nil {
				item.Metadata = meta
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}

