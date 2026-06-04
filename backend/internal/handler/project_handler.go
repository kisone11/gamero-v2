// Package handler 提供项目系统的 HTTP 请求处理层。
// 本文件包含项目相关所有 API 的 Gin Handler 实现。
// Handler 负责：参数绑定与校验、调用 Service 层、统一响应输出。
package handler

import (
	"strconv"
	"time"

	"github.com/gamero/gamero/internal/service"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gamero/gamero/pkg/storage"
	"github.com/gin-gonic/gin"
)

// ProjectHandler 项目相关 API 的 Handler 集合
type ProjectHandler struct {
	svc service.ProjectService
}

// NewProjectHandler 创建 ProjectHandler 实例
func NewProjectHandler(svc service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// ===========================
// 工具函数
// ===========================

// parseProjectID 从路由参数 :id 中解析项目 ID
func parseProjectID(c *gin.Context) (uint64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的项目 ID")
		return 0, false
	}
	return id, true
}

// parseMemberID 从路由参数 :memberID 中解析成员 ID
func parseMemberID(c *gin.Context) (uint64, bool) {
	idStr := c.Param("memberID")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的成员 ID")
		return 0, false
	}
	return id, true
}

func parseTaskID(c *gin.Context) (uint64, bool) {
	idStr := c.Param("taskID")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的任务 ID")
		return 0, false
	}
	return id, true
}

// ===========================
// 项目基本操作（需 JWT，且限 owner）
// ===========================

// CreateProject 创建项目
// POST /api/v1/projects
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req service.CreateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	project, err := h.svc.CreateProject(c.Request.Context(), userID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, project)
}

// ListTasks 获取项目任务列表
func (h *ProjectHandler) ListTasks(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}
	tasks, err := h.svc.ListTasks(c.Request.Context(), userID, projectID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, tasks)
}

// CreateTask 创建项目任务
func (h *ProjectHandler) CreateTask(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}
	var req service.ProjectTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}
	task, err := h.svc.CreateTask(c.Request.Context(), userID, projectID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, task)
}

// UpdateTask 更新项目任务
func (h *ProjectHandler) UpdateTask(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}
	var req service.ProjectTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}
	task, err := h.svc.UpdateTask(c.Request.Context(), userID, projectID, taskID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, task)
}

// DeleteTask 删除项目任务
func (h *ProjectHandler) DeleteTask(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteTask(c.Request.Context(), userID, projectID, taskID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// UpdateProject 更新项目信息（仅 owner）
// PUT /api/v1/projects/:id
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	var req service.UpdateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	project, err := h.svc.UpdateProject(c.Request.Context(), userID, projectID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, project)
}

// DeleteProject 软删除项目（仅 owner）
// DELETE /api/v1/projects/:id
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteProject(c.Request.Context(), userID, projectID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "项目已删除", nil)
}

// UpdateStatus 更新项目状态（仅 owner）
// PUT /api/v1/projects/:id/status
func (h *ProjectHandler) UpdateStatus(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	var req service.UpdateStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	project, err := h.svc.UpdateStatus(c.Request.Context(), userID, projectID, req.Status)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, project)
}

// SaveCoverReq 提交封面 key 请求体
type SaveCoverReq struct {
	Key string `json:"key" binding:"required"`
}

// SaveCover 提交封面 key（前端直传后调用）
// PUT /api/v1/projects/:id/cover
func (h *ProjectHandler) SaveCover(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	var req SaveCoverReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}

	if err := h.svc.SaveCover(c.Request.Context(), userID, projectID, req.Key); err != nil {
		response.Fail(c, err)
		return
	}

	// 返回预签名 GET URL 供前端展示
	presignedURL, _ := storage.PresignedGetURL(c.Request.Context(), req.Key, 24*time.Hour)
	response.Success(c, gin.H{"url": presignedURL, "key": req.Key})
}

// SaveScreenshotReq 提交截图 key 请求体
type SaveScreenshotReq struct {
	Key string `json:"key" binding:"required"`
}

// SaveScreenshot 提交截图 key（前端直传后调用）
// POST /api/v1/projects/:id/screenshots
func (h *ProjectHandler) SaveScreenshot(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	var req SaveScreenshotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}

	if err := h.svc.SaveScreenshot(c.Request.Context(), userID, projectID, req.Key); err != nil {
		response.Fail(c, err)
		return
	}

	// 返回预签名 GET URL 供前端展示
	presignedURL, _ := storage.PresignedGetURL(c.Request.Context(), req.Key, 24*time.Hour)
	response.Success(c, gin.H{"url": presignedURL, "key": req.Key})
}

// DeleteScreenshot 删除截图（仅 owner）
// DELETE /api/v1/projects/:id/screenshots
// 截图 对象存储 key 通过请求体 JSON 传入（key 字段），避免 URL 中斜线路由问题
func (h *ProjectHandler) DeleteScreenshot(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	var body struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FailBadRequest(c, "请在请求体中传入 key 字段（截图 对象存储 key）")
		return
	}

	if err := h.svc.DeleteScreenshot(c.Request.Context(), userID, projectID, body.Key); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "截图已删除", nil)
}

// SaveVideoReq 提交视频 key 请求体
type SaveVideoReq struct {
	Key string `json:"key" binding:"required"`
}

// SaveVideo 提交视频 key（前端直传后调用）
// POST /api/v1/projects/:id/videos
func (h *ProjectHandler) SaveVideo(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	var req SaveVideoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}

	if err := h.svc.SaveVideo(c.Request.Context(), userID, projectID, req.Key); err != nil {
		response.Fail(c, err)
		return
	}

	// 返回预签名 GET URL 供前端展示
	presignedURL, _ := storage.PresignedGetURL(c.Request.Context(), req.Key, 24*time.Hour)
	response.Success(c, gin.H{"url": presignedURL, "key": req.Key})
}

// DeleteVideo 删除视频（仅 owner）
// DELETE /api/v1/projects/:id/videos
func (h *ProjectHandler) DeleteVideo(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
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

	if err := h.svc.DeleteVideo(c.Request.Context(), userID, projectID, body.Key); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "视频已删除", nil)
}

// ===========================
// 项目浏览（公开，可选 JWT）
// ===========================

// ListProjects 项目列表（分页+筛选）
// GET /api/v1/projects
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	pager := pagination.Parse(c)

	sortBy := c.Query("sort_by")
	if sortBy == "" {
		sortBy = c.Query("sort")
	}

	params := service.ListProjectsParams{
		Page:     pager.Page,
		PageSize: pager.PageSize,
		Status:   c.Query("status"),
		Genre:    c.Query("genre"),
		Keyword:  c.Query("keyword"),
		SortBy:   sortBy, // newest（默认）/ hottest / updated
	}
	if ownerID := c.Query("owner_id"); ownerID != "" {
		id, err := strconv.ParseUint(ownerID, 10, 64)
		if err != nil {
			response.FailBadRequest(c, "owner_id 参数错误")
			return
		}
		params.OwnerID = id
	}
	if participantID := c.Query("participant_id"); participantID != "" {
		id, err := strconv.ParseUint(participantID, 10, 64)
		if err != nil {
			response.FailBadRequest(c, "participant_id 参数错误")
			return
		}
		params.ParticipantID = id
	}

	items, total, err := h.svc.ListProjects(c.Request.Context(), params)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// GetProjectByID 获取项目详情（by ID）
// GET /api/v1/projects/:id
func (h *ProjectHandler) GetProjectByID(c *gin.Context) {
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	// 可选 JWT，viewerID 用于后续权限判断扩展
	viewerID, _ := middleware.GetUserID(c)

	detail, err := h.svc.GetProject(c.Request.Context(), projectID, viewerID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, detail)
}

// GetProjectBySlug 获取项目详情（by slug）
// GET /api/v1/projects/slug/:slug
func (h *ProjectHandler) GetProjectBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		response.FailBadRequest(c, "slug 不能为空")
		return
	}

	viewerID, _ := middleware.GetUserID(c)

	detail, err := h.svc.GetProjectBySlug(c.Request.Context(), slug, viewerID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, detail)
}

// GetTimeline 获取项目时间轴（分页）
// GET /api/v1/projects/:id/timeline
func (h *ProjectHandler) GetTimeline(c *gin.Context) {
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	pager := pagination.Parse(c)

	events, total, err := h.svc.GetTimeline(c.Request.Context(), projectID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, events, total, pager.Page, pager.PageSize)
}

// GetProjectMembers 获取项目成员列表（公开接口）
// GET /api/v1/projects/:id/members
func (h *ProjectHandler) GetProjectMembers(c *gin.Context) {
	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	members, err := h.svc.GetProjectMembers(c.Request.Context(), projectID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, members)
}

// ===========================
// 成员管理（需 JWT，且限 owner）
// ===========================

// AddMember 添加项目成员（仅 owner）
// POST /api/v1/projects/:id/members
func (h *ProjectHandler) AddMember(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	var req service.AddMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	member, err := h.svc.AddMember(c.Request.Context(), userID, projectID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, member)
}

// UpdateMember 更新成员角色/贡献（仅 owner）
// PUT /api/v1/projects/:id/members/:memberID
func (h *ProjectHandler) UpdateMember(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	memberID, ok := parseMemberID(c)
	if !ok {
		return
	}

	var req service.UpdateMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	if err := h.svc.UpdateMember(c.Request.Context(), userID, projectID, memberID, &req); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "成员信息已更新", nil)
}

// RemoveMember 移除项目成员（仅 owner）
// DELETE /api/v1/projects/:id/members/:memberID
func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, ok := parseProjectID(c)
	if !ok {
		return
	}

	memberID, ok := parseMemberID(c)
	if !ok {
		return
	}

	if err := h.svc.RemoveMember(c.Request.Context(), userID, projectID, memberID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "成员已移除", nil)
}

// GetUserProjects 获取指定用户名的公开项目列表
// GET /api/v1/users/:username/projects
func (h *ProjectHandler) GetUserProjects(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "用户id不正确")
		return
	}

	pager := pagination.Parse(c)
	items, total, err := h.svc.GetUserProjects(c.Request.Context(), id, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// LeaveProject 成员主动退出项目
// DELETE /api/v1/projects/:id/members/me
func (h *ProjectHandler) LeaveProject(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.LeaveProject(c.Request.Context(), userID, projectID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// TransferOwner 转让项目所有权
// PUT /api/v1/projects/:id/transfer-owner
func (h *ProjectHandler) TransferOwner(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req struct {
		NewOwnerID uint64 `json:"new_owner_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperrors.New(apperrors.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.TransferOwner(c.Request.Context(), userID, projectID, req.NewOwnerID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// CollectProject 收藏项目
// POST /api/v1/projects/:id/collect
func (h *ProjectHandler) CollectProject(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.CollectProject(c.Request.Context(), userID, projectID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// UncollectProject 取消收藏项目
// DELETE /api/v1/projects/:id/collect
func (h *ProjectHandler) UncollectProject(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.UncollectProject(c.Request.Context(), userID, projectID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// GetMyCollectedProjects 获取我收藏的项目
// GET /api/v1/me/collected-projects
func (h *ProjectHandler) GetMyCollectedProjects(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	pager := pagination.Parse(c)
	items, total, err := h.svc.GetMyCollectedProjects(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}
