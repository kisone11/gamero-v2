// Package handler 提供玩家游戏评测的 HTTP Handler 实现。
package handler

import (
	"github.com/gamero/gamero/internal/service"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// ReviewHandler 玩家评测 HTTP 处理器
type ReviewHandler struct {
	svc service.ReviewService
}

// NewReviewHandler 创建 ReviewHandler
func NewReviewHandler(svc service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

// CreateReview 创建评测
// POST /api/v1/projects/:id/reviews
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req service.CreateGameReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperrors.New(apperrors.CodeParamInvalid, err.Error()))
		return
	}
	item, err := h.svc.CreateReview(c.Request.Context(), userID, projectID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, item)
}

// UpdateReview 更新评测
// PUT /api/v1/projects/:id/reviews/:reviewId
func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	reviewID, err := parseUint64Param(c, "reviewId")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req service.CreateGameReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperrors.New(apperrors.CodeParamInvalid, err.Error()))
		return
	}
	item, err := h.svc.UpdateReview(c.Request.Context(), userID, reviewID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, item)
}

// DeleteReview 删除评测
// DELETE /api/v1/projects/:id/reviews/:reviewId
func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	reviewID, err := parseUint64Param(c, "reviewId")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.DeleteReview(c.Request.Context(), userID, reviewID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "删除成功", nil)
}

// ListReviews 获取项目评测列表
// GET /api/v1/projects/:id/reviews
func (h *ReviewHandler) ListReviews(c *gin.Context) {
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	viewerID, _ := middleware.GetUserID(c)
	pager := pagination.Parse(c)
	sortBy := c.DefaultQuery("sort_by", "newest") // newest / helpful
	items, total, err := h.svc.ListReviews(c.Request.Context(), projectID, viewerID, pager.Page, pager.PageSize, sortBy)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// GetRatingSummary 获取评分汇总
// GET /api/v1/projects/:id/reviews/summary
func (h *ReviewHandler) GetRatingSummary(c *gin.Context) {
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	summary, err := h.svc.GetRatingSummary(c.Request.Context(), projectID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, summary)
}

// LikeReview 点赞评测
// POST /api/v1/projects/:id/reviews/:reviewId/like
func (h *ReviewHandler) LikeReview(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	reviewID, err := parseUint64Param(c, "reviewId")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.LikeReview(c.Request.Context(), userID, reviewID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// UnlikeReview 取消点赞
// DELETE /api/v1/projects/:id/reviews/:reviewId/like
func (h *ReviewHandler) UnlikeReview(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	reviewID, err := parseUint64Param(c, "reviewId")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.UnlikeReview(c.Request.Context(), userID, reviewID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// ReplyReview 项目作者回复评测
// POST /api/v1/projects/:id/reviews/:reviewId/reply
func (h *ReviewHandler) ReplyReview(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	reviewID, err := parseUint64Param(c, "reviewId")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		Content string `json:"content" binding:"required,max=1000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperrors.New(apperrors.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.ReplyReview(c.Request.Context(), projectID, reviewID, userID, req.Content); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "回复成功", nil)
}
