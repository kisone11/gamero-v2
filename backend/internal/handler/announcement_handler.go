package handler

import (
	"strconv"
	"time"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AnnouncementHandler struct {
	db *gorm.DB
}

type announcementReq struct {
	Title      string `json:"title" binding:"required,min=2,max=120"`
	Content    string `json:"content" binding:"required,min=2,max=5000"`
	Level      string `json:"level"`
	IsActive   *bool  `json:"is_active"`
	IsPinned   bool   `json:"is_pinned"`
	ExpireDays int    `json:"expire_days"`
}

func NewAnnouncementHandler(db *gorm.DB) *AnnouncementHandler {
	return &AnnouncementHandler{db: db}
}

func (h *AnnouncementHandler) ListPublic(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	now := time.Now()
	var items []model.Announcement
	if err := h.db.WithContext(c.Request.Context()).
		Where("is_active = ? AND published_at <= ? AND (expires_at IS NULL OR expires_at > ?)", true, now, now).
		Order("is_pinned DESC, published_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		response.FailInternal(c)
		return
	}
	response.Success(c, items)
}

func (h *AnnouncementHandler) AdminList(c *gin.Context) {
	p := pagination.Parse(c)
	query := h.db.WithContext(c.Request.Context()).Model(&model.Announcement{})
	if active := c.Query("active"); active == "true" || active == "false" {
		query = query.Where("is_active = ?", active == "true")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.FailInternal(c)
		return
	}
	var items []model.Announcement
	if err := query.Order("created_at DESC").Offset((p.Page - 1) * p.PageSize).Limit(p.PageSize).Find(&items).Error; err != nil {
		response.FailInternal(c)
		return
	}
	response.SuccessPage(c, items, total, p.Page, p.PageSize)
}

func (h *AnnouncementHandler) AdminCreate(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	var req announcementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	level := normalizeAnnouncementLevel(req.Level)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	var expiresAt *time.Time
	if req.ExpireDays > 0 {
		t := time.Now().AddDate(0, 0, req.ExpireDays)
		expiresAt = &t
	}
	item := &model.Announcement{
		Title:       req.Title,
		Content:     req.Content,
		Level:       level,
		IsActive:    isActive,
		IsPinned:    req.IsPinned,
		CreatedBy:   adminID,
		PublishedAt: time.Now(),
		ExpiresAt:   expiresAt,
	}
	if err := h.db.WithContext(c.Request.Context()).Create(item).Error; err != nil {
		response.FailInternal(c)
		return
	}
	response.Success(c, item)
}

func (h *AnnouncementHandler) AdminUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "公告 ID 错误")
		return
	}
	var req announcementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	updates := map[string]interface{}{
		"title":     req.Title,
		"content":   req.Content,
		"level":     normalizeAnnouncementLevel(req.Level),
		"is_active": isActive,
		"is_pinned": req.IsPinned,
	}
	if req.ExpireDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpireDays)
		updates["expires_at"] = &expiresAt
	}
	if err := h.db.WithContext(c.Request.Context()).Model(&model.Announcement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.FailInternal(c)
		return
	}
	response.Success(c, nil)
}

func (h *AnnouncementHandler) AdminDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "公告 ID 错误")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Delete(&model.Announcement{}, id).Error; err != nil {
		response.FailInternal(c)
		return
	}
	response.Success(c, nil)
}

func normalizeAnnouncementLevel(level string) string {
	switch level {
	case "warning", "danger", "success":
		return level
	default:
		return "info"
	}
}
