// Package handler 提供开发日志系统的 HTTP 请求处理层。
// 本文件包含开发日志相关所有 API 的 Gin Handler 实现。
// Handler 负责：参数绑定与校验、调用 Service 层、统一响应输出。
package handler

import (
	"strconv"
	"time"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gamero/gamero/pkg/storage"
	"github.com/gin-gonic/gin"
)

// DevLogHandler 开发日志相关 API 的 Handler 集合
type DevLogHandler struct {
	svc service.DevLogService
}

// NewDevLogHandler 创建 DevLogHandler 实例
func NewDevLogHandler(svc service.DevLogService) *DevLogHandler {
	return &DevLogHandler{svc: svc}
}

// ===========================
// 工具函数
// ===========================

// parseLogID 从路由参数 :logID 中解析日志 ID
func parseLogID(c *gin.Context) (uint64, bool) {
	idStr := c.Param("logID")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的日志 ID")
		return 0, false
	}
	return id, true
}

// parseCommentID 从路由参数 :commentID 中解析评论 ID
func parseCommentID(c *gin.Context) (uint64, bool) {
	idStr := c.Param("commentID")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的评论 ID")
		return 0, false
	}
	return id, true
}

// optionalUserID 从 Gin Context 获取当前用户 ID（未登录返回 0，不报错）
func optionalUserID(c *gin.Context) uint64 {
	userID, _ := middleware.GetUserID(c)
	return userID
}

// ===========================
// 日志基础操作
// ===========================

// CreateLog 创建项目开发日志
// POST /api/v1/projects/:id/logs
func (h *DevLogHandler) CreateLog(c *gin.Context) {
	// 获取登录用户
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	// 解析项目 ID
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	var req service.CreateDevLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数格式错误")
		return
	}

	log, err := h.svc.CreateLog(c.Request.Context(), userID, projectID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, log)
}

// ListLogs 获取项目日志列表（公开，分页）
// GET /api/v1/projects/:id/logs
func (h *DevLogHandler) ListLogs(c *gin.Context) {
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	// 可选 JWT（用于过滤 members_only 日志）
	viewerID := optionalUserID(c)

	pager := pagination.Parse(c)

	params := &service.ListLogsParams{
		ViewerID: viewerID,
		Page:     pager.Page,
		PageSize: pager.PageSize,
	}

	// 查询参数筛选
	if logType := c.Query("log_type"); logType != "" {
		params.LogType = service.DevLogTypeFromString(logType)
	}
	if status := c.Query("status"); status != "" {
		params.Status = service.DevLogStatusFromString(status)
	}

	logs, total, err := h.svc.ListLogs(c.Request.Context(), projectID, params)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, logs, total, pager.Page, pager.PageSize)
}

// GetLog 获取日志详情（公开或成员可见）
// GET /api/v1/logs/:logID
func (h *DevLogHandler) GetLog(c *gin.Context) {
	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	viewerID := optionalUserID(c)

	detail, err := h.svc.GetLog(c.Request.Context(), logID, viewerID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, detail)
}

// UpdateLog 更新日志（需JWT，作者）
// PUT /api/v1/logs/:logID
func (h *DevLogHandler) UpdateLog(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	var req service.UpdateDevLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数格式错误")
		return
	}

	// 敏感词检测
	titleStr := ""
	if req.Title != nil {
		titleStr = *req.Title
	}
	contentStr := ""
	if req.Content != nil {
		contentStr = *req.Content
	}
	if filter.Contains(titleStr) || filter.Contains(contentStr) {
		response.FailBadRequest(c, "内容含有违规词汇，请修改后重试")
		return
	}

	log, err := h.svc.UpdateLog(c.Request.Context(), userID, logID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, log)
}

// DeleteLog 删除日志（需JWT，作者）
// DELETE /api/v1/logs/:logID
func (h *DevLogHandler) DeleteLog(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteLog(c.Request.Context(), userID, logID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "日志已删除", nil)
}

// PublishLog 发布草稿（需JWT，作者）
// POST /api/v1/logs/:logID/publish
func (h *DevLogHandler) PublishLog(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	log, err := h.svc.PublishLog(c.Request.Context(), userID, logID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, log)
}

// ListReleases 版本历史列表（公开）
// GET /api/v1/projects/:id/releases
func (h *DevLogHandler) ListReleases(c *gin.Context) {
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	pager := pagination.Parse(c)

	logs, total, err := h.svc.ListReleases(c.Request.Context(), projectID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, logs, total, pager.Page, pager.PageSize)
}

// SaveLogMediaReq 提交日志媒体 key 请求体
type SaveLogMediaReq struct {
	Key string `json:"key" binding:"required"`
}

// SaveLogImage 提交日志图片 key（前端直传后调用）
// PUT /api/v1/logs/:logID/images
func (h *DevLogHandler) SaveLogImage(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	var req SaveLogMediaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}

	result, err := h.svc.SaveLogImage(c.Request.Context(), userID, logID, req.Key)
	if err != nil {
		response.Fail(c, err)
		return
	}

	// 返回预签名 GET URL 供前端展示
	presignedURL, _ := storage.PresignedGetURL(c.Request.Context(), req.Key, 24*time.Hour)
	response.Success(c, gin.H{"url": presignedURL, "key": result.Key})
}

// SaveLogVideo 提交日志视频 key（前端直传后调用）
// PUT /api/v1/logs/:logID/videos
func (h *DevLogHandler) SaveLogVideo(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	var req SaveLogMediaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}

	result, err := h.svc.SaveLogVideo(c.Request.Context(), userID, logID, req.Key)
	if err != nil {
		response.Fail(c, err)
		return
	}

	// 返回预签名 GET URL 供前端展示
	presignedURL, _ := storage.PresignedGetURL(c.Request.Context(), req.Key, 24*time.Hour)
	response.Success(c, gin.H{"url": presignedURL, "key": result.Key})
}

// DeleteLogImage 删除日志中的单个图片（需JWT，作者）
// DELETE /api/v1/logs/:logID/images
// 请求体 JSON: { "key": "<对象存储 key>" }
func (h *DevLogHandler) DeleteLogImage(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	var body struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FailBadRequest(c, "请在请求体中传入 key 字段（图片 对象存储 key）")
		return
	}

	if err := h.svc.DeleteLogImage(c.Request.Context(), userID, logID, body.Key); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "删除成功", nil)
}

// DeleteLogVideo 删除日志中的单个视频（需JWT，作者）
// DELETE /api/v1/logs/:logID/videos
// 请求体 JSON: { "key": "<对象存储 key>" }
func (h *DevLogHandler) DeleteLogVideo(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	var body struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FailBadRequest(c, "请在请求体中传入 key 字段（视频 对象存储 key）")
		return
	}

	if err := h.svc.DeleteLogVideo(c.Request.Context(), userID, logID, body.Key); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "删除成功", nil)
}

// ===========================
// 日志互动
// ===========================

// LikeLog 点赞日志（需JWT）
// POST /api/v1/logs/:logID/like
func (h *DevLogHandler) LikeLog(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	if err := h.svc.LikeLog(c.Request.Context(), userID, logID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "点赞成功", nil)
}

// UnlikeLog 取消点赞（需JWT）
// DELETE /api/v1/logs/:logID/like
func (h *DevLogHandler) UnlikeLog(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	if err := h.svc.UnlikeLog(c.Request.Context(), userID, logID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "已取消点赞", nil)
}

// CollectLog 收藏日志（需JWT）
// POST /api/v1/logs/:logID/collect
func (h *DevLogHandler) CollectLog(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	if err := h.svc.CollectLog(c.Request.Context(), userID, logID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "收藏成功", nil)
}

// UncollectLog 取消收藏（需JWT）
// DELETE /api/v1/logs/:logID/collect
func (h *DevLogHandler) UncollectLog(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	if err := h.svc.UncollectLog(c.Request.Context(), userID, logID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "已取消收藏", nil)
}

// ===========================
// 评论
// ===========================

// CreateComment 发表评论（需JWT）
// POST /api/v1/logs/:logID/comments
func (h *DevLogHandler) CreateComment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	var req service.CreateCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数格式错误")
		return
	}

	comment, err := h.svc.CreateComment(c.Request.Context(), userID, logID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, comment)
}

// ListComments 评论列表（公开，分页）
// GET /api/v1/logs/:logID/comments
func (h *DevLogHandler) ListComments(c *gin.Context) {
	logID, ok := parseLogID(c)
	if !ok {
		return
	}

	// 可选 JWT（用于显示是否已点赞）
	viewerID := optionalUserID(c)

	sortBy := c.DefaultQuery("sort_by", "latest") // latest or hot

	pager := pagination.Parse(c)

	comments, total, err := h.svc.ListComments(c.Request.Context(), logID, viewerID, pager.Page, pager.PageSize, sortBy)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, comments, total, pager.Page, pager.PageSize)
}

// DeleteComment 删除评论（需JWT，作者或日志作者）
// DELETE /api/v1/comments/:commentID
func (h *DevLogHandler) DeleteComment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	commentID, ok := parseCommentID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteComment(c.Request.Context(), userID, commentID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "评论已删除", nil)
}

// LikeComment 评论点赞（需JWT）
// POST /api/v1/comments/:commentID/like
func (h *DevLogHandler) LikeComment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	commentID, ok := parseCommentID(c)
	if !ok {
		return
	}

	if err := h.svc.LikeComment(c.Request.Context(), userID, commentID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "点赞成功", nil)
}

// UnlikeComment 取消评论点赞（需JWT）
// DELETE /api/v1/comments/:commentID/like
func (h *DevLogHandler) UnlikeComment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	commentID, ok := parseCommentID(c)
	if !ok {
		return
	}

	if err := h.svc.UnlikeComment(c.Request.Context(), userID, commentID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "已取消点赞", nil)
}

// GetMyCollectedLogs 获取当前用户收藏的开发日志列表（需JWT）
// GET /api/v1/me/collections/logs
func (h *DevLogHandler) GetMyCollectedLogs(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	pager := pagination.Parse(c)

	logs, total, err := h.svc.GetMyCollectedLogs(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, logs, total, pager.Page, pager.PageSize)
}

// GetMyLogs 获取当前用户发布的开发日志列表（需JWT）
// GET /api/v1/me/logs
func (h *DevLogHandler) GetMyLogs(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	pager := pagination.Parse(c)

	logs, total, err := h.svc.GetMyLogs(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, logs, total, pager.Page, pager.PageSize)
}
