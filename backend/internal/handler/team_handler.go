// Package handler 提供组队系统的 HTTP 请求处理层。
// 本文件包含招募、申请、合作评价、人才库、通知等全部 API 的 Gin Handler 实现。
// Handler 负责：参数绑定与校验、调用 Service 层、统一响应输出。
package handler

import (
	"strconv"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/service"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// 确保 model 包被引用（用于 GetPresetReviewTags） - 此行可删除，因为 model 在本文件内直接使用

// TeamHandler 组队系统相关 API 的 Handler 集合
type TeamHandler struct {
	svc service.TeamService
}

// NewTeamHandler 创建 TeamHandler 实例
func NewTeamHandler(svc service.TeamService) *TeamHandler {
	return &TeamHandler{svc: svc}
}

// ===========================
// 工具函数（team_handler 内部）
// ===========================

// parseIDParam 通用路由参数解析，解析失败时自动响应 400 并返回 (0, false)。
// 替代原有的 parseRecruitmentID / parseAppID / parseReviewID，统一走 parseUint64Param。
func parseIDParam(c *gin.Context, paramName string, label string) (uint64, bool) {
	id, err := parseUint64Param(c, paramName)
	if err != nil {
		response.FailBadRequest(c, "无效的"+label)
		return 0, false
	}
	return id, true
}

// ===========================
// 招募相关 Handler
// ===========================

// CreateRecruitment 发布招募信息
// POST /api/v1/recruitments
func (h *TeamHandler) CreateRecruitment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req service.CreateRecruitmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	rec, err := h.svc.CreateRecruitment(c.Request.Context(), userID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, rec)
}

// ListRecruitments 招募列表（公开，支持筛选）
// GET /api/v1/recruitments
func (h *TeamHandler) ListRecruitments(c *gin.Context) {
	p := pagination.Parse(c)

	var req service.ListRecruitmentsReq
	req.Page = p.Page
	req.PageSize = p.PageSize

	// 从 query 参数中读取筛选条件
	req.Position = c.Query("position")
	req.CooperationType = c.Query("cooperation_type")
	req.Keyword = c.Query("keyword")

	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if pid, err := strconv.ParseUint(projectIDStr, 10, 64); err == nil {
			req.ProjectID = pid
		}
	}

	if ownerIDStr := c.Query("owner_id"); ownerIDStr != "" {
		if oid, err := strconv.ParseUint(ownerIDStr, 10, 64); err == nil {
			req.OwnerID = oid
		}
	}

	list, total, err := h.svc.ListRecruitments(c.Request.Context(), &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, list, total, p.Page, p.PageSize)
}

// GetRecruitment 招募详情（公开）
// GET /api/v1/recruitments/:id
func (h *TeamHandler) GetRecruitment(c *gin.Context) {
	id, ok := parseIDParam(c, "id", "招募 ID")
	if !ok {
		return
	}

	detail, err := h.svc.GetRecruitment(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, detail)
}

// UpdateRecruitment 更新招募信息（需JWT，owner）
// PUT /api/v1/recruitments/:id
func (h *TeamHandler) UpdateRecruitment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id", "招募 ID")
	if !ok {
		return
	}

	var req service.UpdateRecruitmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	rec, err := h.svc.UpdateRecruitment(c.Request.Context(), userID, id, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, rec)
}

// CloseRecruitment 关闭招募（需JWT，owner）
// POST /api/v1/recruitments/:id/close
func (h *TeamHandler) CloseRecruitment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id", "招募 ID")
	if !ok {
		return
	}

	if err := h.svc.CloseRecruitment(c.Request.Context(), userID, id); err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, gin.H{"message": "招募已关闭"})
}

// ReopenRecruitment 重开招募（需JWT，owner）
// POST /api/v1/recruitments/:id/reopen
func (h *TeamHandler) ReopenRecruitment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id", "招募 ID")
	if !ok {
		return
	}

	if err := h.svc.ReopenRecruitment(c.Request.Context(), userID, id); err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, gin.H{"message": "招募已重新开启"})
}

// GetExpiringSoonRecruitments 查询7天内即将过期的招募（公开）
// GET /api/v1/recruitments/expiring-soon
func (h *TeamHandler) GetExpiringSoonRecruitments(c *gin.Context) {
	items, err := h.svc.ListExpiringSoonRecruitments(c.Request.Context(), 7)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, items)
}

// ===========================
// 申请相关 Handler
// ===========================

// ApplyRecruitment 提交申请（需JWT）
// POST /api/v1/recruitments/:id/apply
func (h *TeamHandler) ApplyRecruitment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	recruitmentID, ok := parseIDParam(c, "id", "招募 ID")
	if !ok {
		return
	}

	var req service.ApplyRecruitmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	app, err := h.svc.ApplyRecruitment(c.Request.Context(), userID, recruitmentID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, app)
}

// WithdrawApplication 撤回申请（需JWT）
// DELETE /api/v1/recruitments/:id/apply
func (h *TeamHandler) WithdrawApplication(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	recruitmentID, ok := parseIDParam(c, "id", "招募 ID")
	if !ok {
		return
	}

	if err := h.svc.WithdrawApplication(c.Request.Context(), userID, recruitmentID); err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, gin.H{"message": "申请已撤回"})
}

// GetMyApplications 我的申请列表（需JWT）
// GET /api/v1/me/applications
func (h *TeamHandler) GetMyApplications(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	p := pagination.Parse(c)

	list, total, err := h.svc.GetMyApplications(c.Request.Context(), userID, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, list, total, p.Page, p.PageSize)
}

// GetRecruitmentApplications 查看招募下的申请列表（需JWT，owner）
// GET /api/v1/recruitments/:id/applications
func (h *TeamHandler) GetRecruitmentApplications(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	recruitmentID, ok := parseIDParam(c, "id", "招募 ID")
	if !ok {
		return
	}

	p := pagination.Parse(c)

	list, total, err := h.svc.GetRecruitmentApplications(c.Request.Context(), userID, recruitmentID, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, list, total, p.Page, p.PageSize)
}

// ApproveApplication 通过申请（需JWT，owner）
// POST /api/v1/applications/:appID/approve
func (h *TeamHandler) ApproveApplication(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	appID, ok := parseIDParam(c, "appID", "申请 ID")
	if !ok {
		return
	}

	app, err := h.svc.HandleApplication(c.Request.Context(), userID, appID, true)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, app)
}

// RejectApplication 拒绝申请（需JWT，owner）
// POST /api/v1/applications/:appID/reject
func (h *TeamHandler) RejectApplication(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	appID, ok := parseIDParam(c, "appID", "申请 ID")
	if !ok {
		return
	}

	app, err := h.svc.HandleApplication(c.Request.Context(), userID, appID, false)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, app)
}

// ===========================
// 合作评价相关 Handler
// ===========================

// CreateReview 发表合作评价（需JWT）
// POST /api/v1/collab-reviews  或  POST /api/v1/projects/:id/collaboration-reviews
func (h *TeamHandler) CreateReview(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	// 优先从路径参数读 projectID（兼容旧路由）
	var projectID uint64
	if idStr := c.Param("id"); idStr != "" {
		pid, err := parseUint64Param(c, "id")
		if err != nil {
			response.FailBadRequest(c, "无效的项目 ID")
			return
		}
		projectID = pid
	}

	var req struct {
		service.CreateReviewReq
		ProjectID uint64 `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}
	if projectID == 0 {
		projectID = req.ProjectID
	}
	if projectID == 0 {
		response.FailBadRequest(c, "project_id 不能为空")
		return
	}

	review, err := h.svc.CreateReview(c.Request.Context(), userID, projectID, &req.CreateReviewReq)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, review)
}

// AddReviewSupplement 补充评价说明（需JWT）
// POST /api/v1/reviews/:reviewID/supplement
func (h *TeamHandler) AddReviewSupplement(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	reviewID, ok := parseIDParam(c, "reviewID", "评价 ID")
	if !ok {
		return
	}

	var req service.AddReviewSupplementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	review, err := h.svc.AddReviewSupplement(c.Request.Context(), userID, reviewID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, review)
}

// GetUserReviews 获取用户收到的评价列表（公开）
// GET /api/v1/users/:id/reviews
func (h *TeamHandler) GetUserReviews(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "用户 ID 格式错误")
		return
	}

	p := pagination.Parse(c)

	list, total, err := h.svc.GetUserReviews(c.Request.Context(), id, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, list, total, p.Page, p.PageSize)
}

// GetUserRating 获取用户评分摘要（公开）
// GET /api/v1/users/:id/rating
func (h *TeamHandler) GetUserRating(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "用户 ID 格式错误")
		return
	}

	summary, err := h.svc.GetUserRatingSummary(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, summary)
}

// ===========================
// 人才库 Handler
// ===========================

// ListTalents 人才库列表（公开）
// GET /api/v1/talents
func (h *TeamHandler) ListTalents(c *gin.Context) {
	p := pagination.Parse(c)

	req := &service.ListTalentsReq{
		Page:           p.Page,
		PageSize:       p.PageSize,
		Category:       c.Query("category"),
		Keyword:        c.Query("keyword"),
		Level:          c.Query("level"),
		CoopPreference: c.Query("coop_preference"),
		Location:       c.Query("location"),
	}
	// only_available=1/true 时只显示开放合作的人才
	req.OnlyAvailable = c.Query("only_available") == "1" || c.Query("only_available") == "true"

	list, total, err := h.svc.ListTalents(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, list, total, p.Page, p.PageSize)
}

// ===========================
// 通知相关 Handler（供用户使用，非内部接口）
// ===========================

// requireTeamLogin 获取当前登录用户 ID（组队 handler 内部使用）
// 复用 project_handler.go 中已定义的 requireLogin 函数，不重复定义
// 注意：requireLogin 已在 project_handler.go 中定义，此处可直接调用

// GetPresetReviewTags 获取预设评价标签列表（公开）
// GET /api/v1/review-tags
func (h *TeamHandler) GetPresetReviewTags(c *gin.Context) {
	// 返回所有预设评价标签
	response.Success(c, gin.H{
		"tags": model.PresetReviewTags,
	})
}

// ===========================
// 人才合作偏好 Handler
// ===========================

// SetAvailability 设置自己是否开放合作（加入/退出人才库）
// PUT /api/v1/me/availability
func (h *TeamHandler) SetAvailability(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	var req service.SetAvailabilityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperrors.New(apperrors.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.SetAvailability(c.Request.Context(), userID, &req); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// ===========================
// 人才邀请 Handler
// ===========================

// InviteTalent 向人才库中的开发者发送邀请
// POST /api/v1/projects/:id/invitations
func (h *TeamHandler) InviteTalent(c *gin.Context) {
	inviterID, ok := requireLogin(c)
	if !ok {
		return
	}
	// 优先从路径参数读 projectID（兼容旧路由 /projects/:id/invitations）
	// 如果路径没有，则从 body 读 project_id（新路由 POST /invitations）
	var projectID uint64
	if idStr := c.Param("id"); idStr != "" {
		pid, err := parseUint64Param(c, "id")
		if err != nil {
			response.Fail(c, err)
			return
		}
		projectID = pid
	}
	var req struct {
		service.InviteTalentReq
		ProjectID uint64 `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperrors.New(apperrors.CodeParamInvalid, err.Error()))
		return
	}
	if projectID == 0 {
		projectID = req.ProjectID
	}
	if projectID == 0 {
		response.Fail(c, apperrors.New(apperrors.CodeParamInvalid, "project_id 不能为空"))
		return
	}
	inv, err := h.svc.InviteTalent(c.Request.Context(), inviterID, projectID, req.TalentID, &req.InviteTalentReq)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, inv)
}

// WithdrawInvitation 撤回邀请
// DELETE /api/v1/invitations/:id
func (h *TeamHandler) WithdrawInvitation(c *gin.Context) {
	inviterID, ok := requireLogin(c)
	if !ok {
		return
	}
	invID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.WithdrawInvitation(c.Request.Context(), inviterID, invID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// RespondInvitation 回应邀请（接受/拒绝）
// POST /api/v1/invitations/:id/respond?action=accept|decline
func (h *TeamHandler) RespondInvitation(c *gin.Context) {
	talentID, ok := requireLogin(c)
	if !ok {
		return
	}
	invID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	// 优先从路径判断 action（新路由 /invitations/:id/accept 或 /decline）
	// 兼容旧路由从 query param 读取
	path := c.Request.URL.Path
	var action string
	switch {
	case len(path) > 7 && path[len(path)-6:] == "accept":
		action = "accept"
	case len(path) > 8 && path[len(path)-7:] == "decline":
		action = "decline"
	default:
		action = c.Query("action")
	}
	if action != "accept" && action != "decline" {
		response.Fail(c, apperrors.New(apperrors.CodeParamInvalid, "action 参数必须为 accept 或 decline"))
		return
	}
	if err := h.svc.RespondInvitation(c.Request.Context(), talentID, invID, action == "accept"); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// GetMyInvitations 查看我收到的邀请列表
// GET /api/v1/me/invitations?status=pending&page=1&page_size=20
func (h *TeamHandler) GetMyInvitations(c *gin.Context) {
	talentID, ok := requireLogin(c)
	if !ok {
		return
	}
	p := pagination.Parse(c)
	status := c.Query("status")
	items, total, err := h.svc.GetMyInvitations(c.Request.Context(), talentID, status, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, p.Page, p.PageSize)
}

// GetProjectInvitations 查看项目发出的邀请列表（项目 owner）
// GET /api/v1/projects/:id/invitations?status=pending&page=1&page_size=20
func (h *TeamHandler) GetProjectInvitations(c *gin.Context) {
	inviterID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	p := pagination.Parse(c)
	status := c.Query("status")
	items, total, err := h.svc.GetProjectInvitations(c.Request.Context(), inviterID, projectID, status, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, p.Page, p.PageSize)
}

// DeleteRecruitment 删除招募（实质为关闭）
// DELETE /api/v1/recruitments/:id
func (h *TeamHandler) DeleteRecruitment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id", "招募 ID")
	if !ok {
		return
	}
	if err := h.svc.CloseRecruitment(c.Request.Context(), userID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}
