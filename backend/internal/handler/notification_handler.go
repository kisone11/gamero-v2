// Package handler 提供 HTTP Handler 实现。
package handler

import (
	"strconv"

	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// NotificationHandler 处理所有通知相关的 HTTP 请求。
// 通知的读/写/标记/删除均通过 NotificationService 统一入口。
type NotificationHandler struct {
	svc      service.NotificationService
	prefRepo repository.NotificationPreferenceRepository
}

// NewNotificationHandler 构造 NotificationHandler。
func NewNotificationHandler(svc service.NotificationService, prefRepo repository.NotificationPreferenceRepository) *NotificationHandler {
	return &NotificationHandler{svc: svc, prefRepo: prefRepo}
}

// GetMyNotifications 获取当前用户的分页通知列表
// GET /api/v1/notifications?page=1&page_size=20&only_unread=true
func (h *NotificationHandler) GetMyNotifications(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	pager := pagination.Parse(c)
	onlyUnread := c.Query("only_unread") == "true"

	notifications, total, err := h.svc.GetMyNotifications(c.Request.Context(), userID, pager.Page, pager.PageSize, onlyUnread)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, notifications, total, pager.Page, pager.PageSize)
}

// MarkNotificationRead 将单条通知标记为已读
// PUT /api/v1/notifications/:nid/read
func (h *NotificationHandler) MarkNotificationRead(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	nid, err := strconv.ParseUint(c.Param("nid"), 10, 64)
	if err != nil || nid == 0 {
		response.FailBadRequest(c, "通知 ID 格式错误")
		return
	}

	if err := h.svc.MarkAsRead(c.Request.Context(), userID, nid); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "标记已读成功", nil)
}

// MarkAllNotificationsRead 将用户所有通知标记为已读
// PUT /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllNotificationsRead(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	if err := h.svc.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "全部已读成功", nil)
}

// GetUnreadNotificationCount 获取未读通知数量（角标用）
// GET /api/v1/notifications/unread-count
func (h *NotificationHandler) GetUnreadNotificationCount(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	count, err := h.svc.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, gin.H{"unread_count": count})
}

// DeleteNotification 删除单条通知（只能删自己的）
// DELETE /api/v1/notifications/:nid
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	nid, err := strconv.ParseUint(c.Param("nid"), 10, 64)
	if err != nil || nid == 0 {
		response.FailBadRequest(c, "无效的通知 ID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, nid); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "删除成功", nil)
}

// ClearNotifications 清空当前用户所有通知
// DELETE /api/v1/notifications
func (h *NotificationHandler) ClearNotifications(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	if err := h.svc.Clear(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "已清空通知", nil)
}

// DeleteReadNotifications 删除当前用户所有已读通知
// DELETE /api/v1/notifications/read
func (h *NotificationHandler) DeleteReadNotifications(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteRead(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "已删除所有已读通知", nil)
}

// GetNotificationCategories 获取各分类未读通知数
// GET /api/v1/notifications/categories
func (h *NotificationHandler) GetNotificationCategories(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	categories, err := h.svc.GetNotificationCategories(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, categories)
}

// GetNotificationsByType 按类型/分类过滤通知列表
// GET /api/v1/notifications/by-type?type=follow&only_unread=true&page=1&page_size=20
func (h *NotificationHandler) GetNotificationsByType(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	notifType := c.Query("type")
	onlyUnread := c.Query("only_unread") == "true"
	pager := pagination.Parse(c)

	items, total, err := h.svc.GetMyNotificationsByType(c.Request.Context(), userID, notifType, pager.Page, pager.PageSize, onlyUnread)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// GetNotificationPreferences 获取当前用户的所有通知偏好
// GET /api/v1/notifications/preferences
func (h *NotificationHandler) GetNotificationPreferences(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	prefs, err := h.prefRepo.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	type prefResp struct {
		Type    string `json:"type"`
		Enabled bool   `json:"enabled"`
	}
	items := make([]prefResp, 0, len(prefs))
	for _, p := range prefs {
		items = append(items, prefResp{Type: p.Type, Enabled: p.Enabled})
	}
	response.Success(c, gin.H{"preferences": items})
}

// UpdateNotificationPreference 更新单条通知偏好
// PATCH /api/v1/notifications/preferences
func (h *NotificationHandler) UpdateNotificationPreference(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req struct {
		Type    string `json:"type" binding:"required"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：缺少 type 字段")
		return
	}

	if err := h.prefRepo.SetPreference(c.Request.Context(), userID, req.Type, req.Enabled); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "更新成功", nil)
}
