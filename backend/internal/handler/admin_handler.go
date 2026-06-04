// Package handler 提供管理后台的 HTTP 请求处理层。
// 本文件包含以下功能的 Handler 实现：
//   - AdminAuthMiddleware：管理员身份校验中间件（需在 JWT 认证后使用）
//   - 用户管理：列表/详情/封禁/认证级别/删除
//   - 内容审核：举报列表/处理举报/强制删帖/强制删评论
//   - 话题管理：创建/更新/删除/上传图标
//   - 数据统计：概览/时序
//
// 所有接口路径前缀为 /api/v1/admin/，均需 JWT 认证 + 管理员校验。
package handler

import (
	"net/http"
	"strconv"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// AdminHandler 管理后台相关 API 的 Handler 集合
type AdminHandler struct {
	svc service.AdminService
}

// NewAdminHandler 创建 AdminHandler 实例
func NewAdminHandler(svc service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ===========================
// 中间件
// ===========================

// AdminAuthMiddleware 管理员身份校验中间件
// 必须在 JWTAuth 中间件之后使用。
// 从 gin.Context 中取出 JWT 解析后的 role 字段，若非 admin 或 superadmin 则返回 403。
func (h *AdminHandler) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := middleware.GetUserRole(c)
		if !ok {
			response.FailUnauthorized(c)
			c.Abort()
			return
		}
		if !middleware.RoleAtLeast(role, middleware.RoleAdmin) {
			response.FailForbidden(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// ===========================
// 工具函数
// ===========================

// parseAdminID 解析路由参数 :id 为 uint64
func parseAdminID(c *gin.Context) (uint64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的 ID 参数")
		return 0, false
	}
	return id, true
}

// requireAdminUser 从 context 获取当前管理员 ID
func requireAdminUser(c *gin.Context) (uint64, bool) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.FailUnauthorized(c)
		return 0, false
	}
	return adminID, true
}

// ===========================
// 3.10.1 用户管理
// ===========================

// ListUsers 获取用户列表
// GET /api/v1/admin/users?keyword=&page=&page_size=
func (h *AdminHandler) ListUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	if len([]rune(keyword)) > 100 {
		response.FailBadRequest(c, "关键词过长（最多100字）")
		return
	}
	p := pagination.Parse(c)

	users, total, err := h.svc.ListUsers(c.Request.Context(), keyword, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, users, total, p.Page, p.PageSize)
}

// GetUser 获取用户详情
// GET /api/v1/admin/users/:id
func (h *AdminHandler) GetUser(c *gin.Context) {
	id, ok := parseAdminID(c)
	if !ok {
		return
	}

	detail, err := h.svc.GetUser(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, detail)
}

// banUserRequest 封禁/解封用户请求体
type banUserRequest struct {
	Banned        bool `json:"banned"`         // true=封禁，false=解封
	DurationHours int  `json:"duration_hours"` // 封号时长（小时），0 表示永久封号
}

// BanUser 封禁或解封用户
// PUT /api/v1/admin/users/:id/ban
func (h *AdminHandler) BanUser(c *gin.Context) {
	id, ok := parseAdminID(c)
	if !ok {
		return
	}

	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	var req banUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}
	if !req.Banned {
		req.Banned = true
	}

	if err := h.svc.BanUser(c.Request.Context(), adminID, id, req.Banned, req.DurationHours); err != nil {
		response.Fail(c, err)
		return
	}

	msg := "封禁成功"
	if !req.Banned {
		msg = "解封成功"
	}
	response.SuccessMsg(c, msg, nil)
}

// BatchBanUsers 批量封禁用户
// POST /api/v1/admin/users/batch-ban
func (h *AdminHandler) BatchBanUsers(c *gin.Context) {
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}
	var req struct {
		UserIDs       []uint64 `json:"user_ids" binding:"required,min=1,max=100"`
		DurationHours int      `json:"duration_hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}
	if err := h.svc.BatchBanUsers(c.Request.Context(), adminID, req.UserIDs, req.DurationHours); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "批量封禁成功", nil)
}

// UnbanUser 解封用户
// DELETE /api/v1/admin/users/:id/ban
func (h *AdminHandler) UnbanUser(c *gin.Context) {
	id, ok := parseAdminID(c)
	if !ok {
		return
	}
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}
	if err := h.svc.BanUser(c.Request.Context(), adminID, id, false, 0); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "解封成功", nil)
}

// DeleteUser 软删除用户
// DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, ok := parseAdminID(c)
	if !ok {
		return
	}

	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	if err := h.svc.AdminDeleteUser(c.Request.Context(), adminID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "用户删除成功", nil)
}

// ===========================
// 3.10.2 内容审核
// ===========================

// ListReports 获取举报列表
// GET /api/v1/admin/reports?target_type=&status=&page=&page_size=
func (h *AdminHandler) ListReports(c *gin.Context) {
	targetType := c.Query("target_type")
	status := c.Query("status")
	p := pagination.Parse(c)

	reports, total, err := h.svc.ListReports(c.Request.Context(), targetType, status, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, reports, total, p.Page, p.PageSize)
}

// handleReportRequest 处理举报请求体
type handleReportRequest struct {
	Status        string `json:"status" binding:"required"` // handled / rejected
	Note          string `json:"note"`                      // 处理备注
	DeleteContent bool   `json:"delete_content"`            // 是否同步删除被举报内容（仅 handled 有效）
}

// HandleReport 处理举报
// PUT /api/v1/admin/reports/:id
func (h *AdminHandler) HandleReport(c *gin.Context) {
	id, ok := parseAdminID(c)
	if !ok {
		return
	}

	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	var req handleReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	if err := h.svc.HandleReport(c.Request.Context(), adminID, id, req.Status, req.Note, req.DeleteContent); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "举报处理成功", nil)
}

// AdminDeletePost 强制删除帖子
// DELETE /api/v1/admin/posts/:id
func (h *AdminHandler) AdminDeletePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的帖子 ID")
		return
	}

	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	if err := h.svc.AdminDeletePost(c.Request.Context(), adminID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "帖子删除成功", nil)
}

// AdminDeleteComment 强制删除评论
// DELETE /api/v1/admin/comments/:id?type=post|log
func (h *AdminHandler) AdminDeleteComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的评论 ID")
		return
	}

	commentType := c.DefaultQuery("type", "post") // 默认处理帖子评论

	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	if err := h.svc.AdminDeleteComment(c.Request.Context(), adminID, id, commentType); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "评论删除成功", nil)
}

// AdminDeleteLog 强制删除日志
// PATCH /api/v1/admin/logs/:id/hide
func (h *AdminHandler) AdminDeleteLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的日志 ID")
		return
	}

	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	if err := h.svc.AdminDeleteLog(c.Request.Context(), adminID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "日志删除成功", nil)
}

// AdminCloseRecruitment 管理员强制关闭招募
// DELETE /api/v1/admin/recruitments/:id
func (h *AdminHandler) AdminCloseRecruitment(c *gin.Context) {
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}
	id, ok := parseAdminID(c)
	if !ok {
		return
	}
	if err := h.svc.AdminCloseRecruitment(c.Request.Context(), adminID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "招募已关闭", nil)
}

// AdminDeleteProject 管理员软删除项目
// DELETE /api/v1/admin/projects/:id
func (h *AdminHandler) AdminDeleteProject(c *gin.Context) {
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}
	id, ok := parseAdminID(c)
	if !ok {
		return
	}
	if err := h.svc.AdminDeleteProject(c.Request.Context(), adminID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "项目删除成功", nil)
}

// ListProjects 获取项目列表
// GET /api/v1/admin/projects?keyword=&page=&page_size=
func (h *AdminHandler) ListProjects(c *gin.Context) {
	keyword := c.Query("keyword")
	p := pagination.Parse(c)
	projects, total, err := h.svc.ListProjects(c.Request.Context(), keyword, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, projects, total, p.Page, p.PageSize)
}

// ListPosts 获取帖子列表
// GET /api/v1/admin/posts?keyword=&page=&page_size=
func (h *AdminHandler) ListPosts(c *gin.Context) {
	keyword := c.Query("keyword")
	p := pagination.Parse(c)
	posts, total, err := h.svc.ListPosts(c.Request.Context(), keyword, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, posts, total, p.Page, p.PageSize)
}

// ListLogs 获取开发日志列表
// GET /api/v1/admin/logs?keyword=&page=&page_size=
func (h *AdminHandler) ListLogs(c *gin.Context) {
	keyword := c.Query("keyword")
	p := pagination.Parse(c)
	logs, total, err := h.svc.ListLogs(c.Request.Context(), keyword, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, logs, total, p.Page, p.PageSize)
}

// ===========================
// 3.10.3 话题管理
// ===========================

// AdminCreateTopic 创建话题
// POST /api/v1/admin/topics
func (h *AdminHandler) AdminCreateTopic(c *gin.Context) {
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	var req service.AdminCreateTopicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	topic, err := h.svc.CreateTopic(c.Request.Context(), adminID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "话题创建成功", topic)
}

// AdminUpdateTopic 更新话题
// PUT /api/v1/admin/topics/:id
func (h *AdminHandler) AdminUpdateTopic(c *gin.Context) {
	id, ok := parseAdminID(c)
	if !ok {
		return
	}

	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	var req service.AdminUpdateTopicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数错误："+err.Error())
		return
	}

	topic, err := h.svc.UpdateTopic(c.Request.Context(), adminID, id, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, topic)
}

// AdminDeleteTopic 软删除话题
// DELETE /api/v1/admin/topics/:id
func (h *AdminHandler) AdminDeleteTopic(c *gin.Context) {
	id, ok := parseAdminID(c)
	if !ok {
		return
	}

	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteTopic(c.Request.Context(), adminID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "话题删除成功", nil)
}

// SaveTopicIconReq 提交话题图标 key 请求体
type SaveTopicIconReq struct {
	Key string `json:"key" binding:"required"`
}

// SaveTopicIcon 提交话题图标 key（前端已直传云存储后调用）
// PUT /api/v1/admin/topics/:id/icon
func (h *AdminHandler) SaveTopicIcon(c *gin.Context) {
	id, ok := parseAdminID(c)
	if !ok {
		return
	}
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}
	var req SaveTopicIconReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}
	if err := h.svc.SaveTopicIcon(c.Request.Context(), adminID, id, req.Key); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, gin.H{"key": req.Key})
}

// ListTopics 获取话题列表
// GET /api/v1/admin/topics
func (h *AdminHandler) ListTopics(c *gin.Context) {
	topics, err := h.svc.ListTopics(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, topics)
}

// ===========================
// 3.10.4 数据统计
// ===========================

// GetPlatformStats 获取平台概览统计数据
// GET /api/v1/admin/stats
func (h *AdminHandler) GetPlatformStats(c *gin.Context) {
	stats, err := h.svc.GetPlatformStats(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, stats)
}

// GetDailyStats 获取每日数据（用于图表）
// GET /api/v1/admin/stats/daily?days=30
func (h *AdminHandler) GetDailyStats(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30
	}

	result, err := h.svc.GetDailyStats(c.Request.Context(), days)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, result)
}

// ===========================
// 3.10.7 审计日志
// ===========================

// GetAuditLogs 查询管理员操作审计日志（分页）
// GET /api/v1/admin/audit-logs?page=1&page_size=20&admin_id=xxx
func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	p := pagination.Parse(c)

	var adminID uint64
	if adminIDStr := c.Query("admin_id"); adminIDStr != "" {
		id, err := strconv.ParseUint(adminIDStr, 10, 64)
		if err != nil {
			response.FailBadRequest(c, "admin_id 参数无效")
			return
		}
		adminID = id
	}

	logs, total, err := h.svc.GetAuditLogs(c.Request.Context(), adminID, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, logs, total, p.Page, p.PageSize)
}

// ExportUsers 导出用户列表 CSV
// GET /api/v1/admin/users/export
// 查询参数：keyword（可选，同 ListUsers 的关键词过滤）
func (h *AdminHandler) ExportUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	if len([]rune(keyword)) > 100 {
		response.FailBadRequest(c, "关键词过长（最多100字）")
		return
	}
	data, err := h.svc.ExportUsers(c.Request.Context(), keyword)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=users.csv")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

// BanProject 管理员下架项目
// PUT /api/v1/admin/projects/:id/ban
func (h *AdminHandler) BanProject(c *gin.Context) {
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}
	projectID, ok := parseAdminID(c)
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数格式错误")
		return
	}

	if err := h.svc.BanProject(c.Request.Context(), adminID, projectID, req.Reason); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "项目已下架", nil)
}

// UnbanProject 管理员恢复项目上架
// DELETE /api/v1/admin/projects/:id/ban
func (h *AdminHandler) UnbanProject(c *gin.Context) {
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}
	projectID, ok := parseAdminID(c)
	if !ok {
		return
	}
	if err := h.svc.UnbanProject(c.Request.Context(), adminID, projectID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "项目已恢复上架", nil)
}

// ===========================
// 敏感词管理
// ===========================

// GetSensitiveWords 获取当前敏感词列表
// GET /api/v1/admin/sensitive-words
func (h *AdminHandler) GetSensitiveWords(c *gin.Context) {
	words := h.svc.GetSensitiveWords()
	response.Success(c, gin.H{"words": words, "total": len(words)})
}

// AddSensitiveWords 批量添加敏感词
// POST /api/v1/admin/sensitive-words
// Body: {"words": ["词1", "词2"]}
func (h *AdminHandler) AddSensitiveWords(c *gin.Context) {
	adminID, ok := requireAdminUser(c)
	if !ok {
		return
	}
	var req struct {
		Words []string `json:"words" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误："+err.Error())
		return
	}
	h.svc.AddSensitiveWords(c.Request.Context(), adminID, req.Words)
	response.SuccessMsg(c, "敏感词已添加", gin.H{"added": len(req.Words)})
}

// DeleteSensitiveWord 删除一个敏感词
// DELETE /api/v1/admin/sensitive-words/:word
func (h *AdminHandler) DeleteSensitiveWord(c *gin.Context) {
	word := c.Param("word")
	if word == "" {
		response.FailBadRequest(c, "词语不能为空")
		return
	}
	h.svc.RemoveSensitiveWord(c.Request.Context(), word)
	response.SuccessMsg(c, "敏感词已删除", nil)
}
