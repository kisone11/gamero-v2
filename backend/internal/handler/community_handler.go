// Package handler 提供社区系统的 HTTP 请求处理层。
// 本文件包含社区相关所有 API 的 Gin Handler 实现。
// Handler 负责：参数绑定与校验、调用 Service 层、统一响应输出。
package handler

import (
	"strconv"
	"strings"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// CommunityHandler 社区相关 API 的 Handler 集合
type CommunityHandler struct {
	svc service.CommunityService
}

// NewCommunityHandler 创建 CommunityHandler 实例
func NewCommunityHandler(svc service.CommunityService, _ ...interface{}) *CommunityHandler {
	return &CommunityHandler{svc: svc}
}

// ===========================
// 工具函数
// ===========================

// parsePostID 从路由参数 :id 中解析帖子 ID
func parsePostID(c *gin.Context) (uint64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的帖子 ID")
		return 0, false
	}
	return id, true
}

// parsePostCommentID 从路由参数 :commentID 中解析帖子评论 ID
func parsePostCommentID(c *gin.Context) (uint64, bool) {
	idStr := c.Param("commentID")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailBadRequest(c, "无效的评论 ID")
		return 0, false
	}
	return id, true
}

// ===========================
// 话题
// ===========================

// GetTopics 获取话题列表（公开，全量，不分页）
// GET /api/v1/topics
func (h *CommunityHandler) GetTopics(c *gin.Context) {
	topics, err := h.svc.GetTopics(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, topics)
}

// GetTopic 获取单个话题详情（公开）
// GET /api/v1/topics/:id
func (h *CommunityHandler) GetTopic(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "无效的话题 ID")
		return
	}
	topic, err := h.svc.GetTopic(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, topic)
}

// ===========================
// 帖子基础操作
// ===========================

// CreatePost 发布帖子（需JWT）
// POST /api/v1/posts
func (h *CommunityHandler) CreatePost(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req service.CreatePostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数格式错误")
		return
	}

	// 敏感词过滤：标题 + 内容
	if filter.Contains(req.Title) || filter.Contains(req.Content) {
		response.FailBadRequest(c, "内容包含违规词汇，请修改后重新提交")
		return
	}

	detail, err := h.svc.CreatePost(c.Request.Context(), userID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, detail)
}

// ListPosts 帖子列表（公开，分页）
// GET /api/v1/posts?topic_id=&project_id=&sort=hot&page=&page_size=
func (h *CommunityHandler) ListPosts(c *gin.Context) {
	// 可选 JWT（用于查询是否已点赞/收藏）
	viewerID := optionalUserID(c)

	pager := pagination.Parse(c)
	params := &service.ListPostsParams{
		ViewerID: viewerID,
		Page:     pager.Page,
		PageSize: pager.PageSize,
	}

	// 解析 topic_id 筛选
	if topicIDStr := c.Query("topic_id"); topicIDStr != "" {
		if tid, err := strconv.ParseUint(topicIDStr, 10, 64); err == nil {
			params.TopicID = tid
		}
	}

	// 解析 project_id 筛选
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if pid, err := strconv.ParseUint(projectIDStr, 10, 64); err == nil {
			params.ProjectID = pid
		}
	}

	// 解析排序方式：sort=hot 按点赞数倒序
	if sort := c.Query("sort"); sort == "hot" {
		params.SortBy = "hot"
	}

	// 解析关键词搜索
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		params.Keyword = q
	}

	items, total, err := h.svc.ListPosts(c.Request.Context(), params)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// GetPost 获取帖子详情（公开）
// GET /api/v1/posts/:id
func (h *CommunityHandler) GetPost(c *gin.Context) {
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	viewerID := optionalUserID(c)

	detail, err := h.svc.GetPost(c.Request.Context(), postID, viewerID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, detail)
}

// UpdatePost 更新帖子（需JWT，作者）
// PUT /api/v1/posts/:id
func (h *CommunityHandler) UpdatePost(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	var req service.UpdatePostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数格式错误")
		return
	}

	// 敏感词检测（与 CreatePost 保持一致）
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

	detail, err := h.svc.UpdatePost(c.Request.Context(), userID, postID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, detail)
}

// DeletePost 删除帖子（需JWT，作者）
// DELETE /api/v1/posts/:id
func (h *CommunityHandler) DeletePost(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	if err := h.svc.DeletePost(c.Request.Context(), userID, postID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "帖子已删除", nil)
}

// SavePostMediaReq 提交帖子媒体 key 请求体
type SavePostMediaReq struct {
	Key string `json:"key" binding:"required"`
}

// SavePostImage 提交帖子图片 key（前端已直传云存储后调用）
// PUT /api/v1/posts/:id/images
func (h *CommunityHandler) SavePostImage(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	var req SavePostMediaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}
	result, err := h.svc.SavePostImage(c.Request.Context(), userID, postID, req.Key)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, result)
}

// SavePostVideo 提交帖子视频 key（前端已直传云存储后调用）
// PUT /api/v1/posts/:id/videos
func (h *CommunityHandler) SavePostVideo(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	var req SavePostMediaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误：需要 key 字段")
		return
	}
	result, err := h.svc.SavePostVideo(c.Request.Context(), userID, postID, req.Key)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, result)
}

// DeletePostVideo 删除帖子中的单个视频（需JWT，作者）
// DELETE /api/v1/posts/:id/videos
// 请求体 JSON: { "key": "<对象存储 key>" }
func (h *CommunityHandler) DeletePostVideo(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
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

	if err := h.svc.DeletePostVideo(c.Request.Context(), userID, postID, body.Key); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "删除成功", nil)
}

// DeletePostImage 删除帖子中的单个图片（需JWT，作者）
// DELETE /api/v1/posts/:id/images
// 请求体 JSON: { "key": "<对象存储 key>" }
func (h *CommunityHandler) DeletePostImage(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	var body struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FailBadRequest(c, "请在请求体中传入 key 字段（图片对象存储 key）")
		return
	}

	if err := h.svc.DeletePostImage(c.Request.Context(), userID, postID, body.Key); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "删除成功", nil)
}

// ===========================
// 帖子互动
// ===========================

// LikePost 点赞帖子（需JWT）
// POST /api/v1/posts/:id/like
func (h *CommunityHandler) LikePost(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	if err := h.svc.LikePost(c.Request.Context(), userID, postID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "点赞成功", nil)
}

// UnlikePost 取消点赞帖子（需JWT）
// DELETE /api/v1/posts/:id/like
func (h *CommunityHandler) UnlikePost(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	if err := h.svc.UnlikePost(c.Request.Context(), userID, postID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "已取消点赞", nil)
}

// CollectPost 收藏帖子（需JWT）
// POST /api/v1/posts/:id/collect
func (h *CommunityHandler) CollectPost(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	if err := h.svc.CollectPost(c.Request.Context(), userID, postID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "收藏成功", nil)
}

// UncollectPost 取消收藏帖子（需JWT）
// DELETE /api/v1/posts/:id/collect
func (h *CommunityHandler) UncollectPost(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	if err := h.svc.UncollectPost(c.Request.Context(), userID, postID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "已取消收藏", nil)
}

// ===========================
// 评论
// ===========================

// CreatePostComment 发表帖子评论（需JWT）
// POST /api/v1/posts/:id/comments
func (h *CommunityHandler) CreatePostComment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	var req service.CreatePostCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数格式错误")
		return
	}

	// 敏感词过滤：评论内容
	if filter.Contains(req.Content) {
		response.FailBadRequest(c, "内容包含违规词汇，请修改后重新提交")
		return
	}

	comment, err := h.svc.CreateComment(c.Request.Context(), userID, postID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, comment)
}

// ListPostComments 帖子评论列表（公开，分页，树形结构）
// GET /api/v1/posts/:id/comments
func (h *CommunityHandler) ListPostComments(c *gin.Context) {
	postID, ok := parsePostID(c)
	if !ok {
		return
	}

	// 可选 JWT（用于显示是否已点赞）
	viewerID := optionalUserID(c)

	sortBy := c.DefaultQuery("sort_by", "latest") // latest or hot

	pager := pagination.Parse(c)

	comments, total, err := h.svc.ListComments(c.Request.Context(), postID, viewerID, pager.Page, pager.PageSize, sortBy)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, comments, total, pager.Page, pager.PageSize)
}

// DeletePostComment 删除帖子评论（需JWT，评论作者或帖子作者）
// DELETE /api/v1/post-comments/:commentID
func (h *CommunityHandler) DeletePostComment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	commentID, ok := parsePostCommentID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteComment(c.Request.Context(), userID, commentID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "评论已删除", nil)
}

// LikePostComment 帖子评论点赞（需JWT）
// POST /api/v1/post-comments/:commentID/like
func (h *CommunityHandler) LikePostComment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	commentID, ok := parsePostCommentID(c)
	if !ok {
		return
	}

	if err := h.svc.LikeComment(c.Request.Context(), userID, commentID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "点赞成功", nil)
}

// UnlikePostComment 取消帖子评论点赞（需JWT）
// DELETE /api/v1/post-comments/:commentID/like
func (h *CommunityHandler) UnlikePostComment(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	commentID, ok := parsePostCommentID(c)
	if !ok {
		return
	}

	if err := h.svc.UnlikeComment(c.Request.Context(), userID, commentID); err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessMsg(c, "已取消点赞", nil)
}

// ===========================
// 用户帖子列表
// ===========================

// GetUserPosts 查询用户发布的帖子（公开，分页）
// GET /api/v1/users/:id/posts
func (h *CommunityHandler) GetUserPosts(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "用户id不正确")
		return
	}

	// 可选 JWT（用于显示是否已点赞/收藏）
	viewerID := optionalUserID(c)
	pager := pagination.Parse(c)

	// 传入 AuthorUsername，service 层内部通过 userRepo 解析为 AuthorID
	// 不再将 repository 层对象注入 handler
	items, total, err := h.svc.ListPosts(c.Request.Context(), &service.ListPostsParams{
		AuthorID: id,
		ViewerID: viewerID,
		Page:     pager.Page,
		PageSize: pager.PageSize,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// ===========================
// 举报
// ===========================

// CreateReport 提交内容举报（需JWT）
// POST /api/v1/reports
func (h *CommunityHandler) CreateReport(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	var req service.CreateReportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "请求参数格式错误")
		return
	}

	report, err := h.svc.CreateReport(c.Request.Context(), userID, &req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, report)
}

// GetMyReports 查询我的举报记录（需JWT，分页）
// GET /api/v1/me/reports
func (h *CommunityHandler) GetMyReports(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	pager := pagination.Parse(c)

	reports, total, err := h.svc.GetMyReports(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, reports, total, pager.Page, pager.PageSize)
}

// GetMyCollectedPosts 获取当前用户收藏的帖子列表（需JWT）
// GET /api/v1/me/collections/posts
func (h *CommunityHandler) GetMyCollectedPosts(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	pager := pagination.Parse(c)

	posts, total, err := h.svc.GetMyCollectedPosts(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, posts, total, pager.Page, pager.PageSize)
}

// GetMyPosts 获取当前用户发布的帖子列表（需JWT）
// GET /api/v1/me/posts
func (h *CommunityHandler) GetMyPosts(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	pager := pagination.Parse(c)

	posts, total, err := h.svc.GetMyPosts(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.SuccessPage(c, posts, total, pager.Page, pager.PageSize)
}
