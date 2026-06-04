// Package handler 提供关注与通知系统的 HTTP 处理器。
// 本文件包含以下 API Handler：
//   - 用户关注/取关/粉丝列表/关注列表
//   - 项目关注/取关/项目关注者列表
//   - 动态流
//   - 话题关注/取关/查询
package handler

import (
	"strconv"

	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/pagination"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// FollowHandler 关注与通知系统的 Handler 集合
type FollowHandler struct {
	svc      service.FollowService
	userRepo repository.UserRepository
}

// NewFollowHandler 创建 FollowHandler 实例
func NewFollowHandler(svc service.FollowService, userRepo repository.UserRepository) *FollowHandler {
	return &FollowHandler{svc: svc, userRepo: userRepo}
}

// ==================== 用户关注 Handler ====================

// FollowUser 关注用户
// POST /api/v1/users/:id/follow
// 需要 JWT 认证。不能关注自己，重复关注返回 409。
func (h *FollowHandler) FollowUser(c *gin.Context) {
	// 获取当前登录用户 ID
	followerID, ok := requireLogin(c)
	if !ok {
		return
	}

	followeeID, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "用户id不正确")
		return
	} else {
		if followeeID <= 0 {
			response.FailBadRequest(c, "用户id不能为0")
			return
		}
	}

	if err := h.svc.FollowUser(c.Request.Context(), followerID, followeeID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "关注成功", nil)
}

// UnfollowUser 取关用户
// DELETE /api/v1/users/:id/follow
// 需要 JWT 认证。
func (h *FollowHandler) UnfollowUser(c *gin.Context) {
	followerID, ok := requireLogin(c)
	if !ok {
		return
	}

	followeeID, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "用户id不正确")
		return
	} else {
		if followeeID <= 0 {
			response.FailBadRequest(c, "用户id不能为0")
			return
		}
	}

	if err := h.svc.UnfollowUser(c.Request.Context(), followerID, followeeID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "取关成功", nil)
}

// GetFollowers 获取用户的粉丝列表（公开，分页）
// GET /api/v1/users/:id/followers?page=1&page_size=20
func (h *FollowHandler) GetFollowers(c *gin.Context) {
	raw := c.Param("id")
	var userID uint64
	if uid, parseErr := strconv.ParseUint(raw, 10, 64); parseErr == nil {
		userID = uid
	} else {
		// 尝试按 username 查找
		user, err := h.userRepo.GetUserByUsername(c.Request.Context(), raw)
		if err != nil || user == nil {
			response.FailBadRequest(c, "用户不存在")
			return
		}
		userID = user.ID
	}
	if userID == 0 {
		response.FailBadRequest(c, "用户id不能为0")
		return
	}

	pager := pagination.Parse(c)
	items, total, err := h.svc.GetFollowers(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// GetFollowing 获取用户的关注列表（公开，分页）
// GET /api/v1/users/:id/following?page=1&page_size=20
func (h *FollowHandler) GetFollowing(c *gin.Context) {
	raw := c.Param("id")
	var userID uint64
	if uid, parseErr := strconv.ParseUint(raw, 10, 64); parseErr == nil {
		userID = uid
	} else {
		user, err := h.userRepo.GetUserByUsername(c.Request.Context(), raw)
		if err != nil || user == nil {
			response.FailBadRequest(c, "用户不存在")
			return
		}
		userID = user.ID
	}
	if userID == 0 {
		response.FailBadRequest(c, "用户id不能为0")
		return
	}

	pager := pagination.Parse(c)
	items, total, err := h.svc.GetFollowing(c.Request.Context(), userID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// ==================== 项目关注 Handler ====================

// FollowProject 关注项目
// POST /api/v1/projects/:id/follow
// 需要 JWT 认证。重复关注返回 409。
func (h *FollowHandler) FollowProject(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "项目 ID 格式错误")
		return
	}

	if err := h.svc.FollowProject(c.Request.Context(), userID, projectID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "关注项目成功", nil)
}

// UnfollowProject 取关项目
// DELETE /api/v1/projects/:id/follow
// 需要 JWT 认证。
func (h *FollowHandler) UnfollowProject(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "项目 ID 格式错误")
		return
	}

	if err := h.svc.UnfollowProject(c.Request.Context(), userID, projectID); err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessMsg(c, "取关项目成功", nil)
}

// GetProjectFollowers 获取项目的关注者列表（公开，分页）
// GET /api/v1/projects/:id/followers?page=1&page_size=20
func (h *FollowHandler) GetProjectFollowers(c *gin.Context) {
	projectID, err := parseUint64Param(c, "id")
	if err != nil {
		response.FailBadRequest(c, "项目 ID 格式错误")
		return
	}

	pager := pagination.Parse(c)
	items, total, err := h.svc.GetProjectFollowers(c.Request.Context(), projectID, pager.Page, pager.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, items, total, pager.Page, pager.PageSize)
}

// ==================== 动态流 Handler ====================

// GetFeed 获取当前用户的动态流（游标分页）
// GET /api/v1/feed?page_size=20&cursor=<unix_ts>
func (h *FollowHandler) GetFeed(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}

	pageSize := pagination.ParsePageSize(c, 20)
	cursorStr := c.Query("cursor")
	var cursor int64
	if cursorStr != "" {
		cursor, _ = strconv.ParseInt(cursorStr, 10, 64)
	}

	items, nextCursor, err := h.svc.GetFeedByCursor(c.Request.Context(), userID, cursor, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
	})
}

// ==================== 话题关注 Handler ====================

// FollowTopic 关注话题
// POST /api/v1/topics/:id/follow
func (h *FollowHandler) FollowTopic(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	topicID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.FollowTopic(c.Request.Context(), userID, topicID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// UnfollowTopic 取关话题
// DELETE /api/v1/topics/:id/follow
func (h *FollowHandler) UnfollowTopic(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	topicID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.svc.UnfollowTopic(c.Request.Context(), userID, topicID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// IsFollowingTopic 查询是否已关注某话题
// GET /api/v1/topics/:id/follow
func (h *FollowHandler) IsFollowingTopic(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	topicID, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	following, err := h.svc.IsFollowingTopic(c.Request.Context(), userID, topicID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, gin.H{"following": following})
}

// GetMyFollowedTopics 获取我关注的话题列表
// GET /api/v1/users/me/followed-topics?page=1&page_size=20
func (h *FollowHandler) GetMyFollowedTopics(c *gin.Context) {
	userID, ok := requireLogin(c)
	if !ok {
		return
	}
	p := pagination.Parse(c)
	topics, total, err := h.svc.GetFollowedTopics(c.Request.Context(), userID, p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.SuccessPage(c, topics, total, p.Page, p.PageSize)
}
