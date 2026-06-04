// Package handler — 角色管理 Handler。
// 提供管理员修改用户角色的接口（仅 superadmin 可操作）
// 以及用户查看自己角色的接口。
package handler

import (
	"strconv"

	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// RoleHandler 角色管理处理器
type RoleHandler struct {
	userSvc service.UserService
}

// NewRoleHandler 创建 RoleHandler 实例
func NewRoleHandler(userSvc service.UserService) *RoleHandler {
	return &RoleHandler{userSvc: userSvc}
}

// setUserRoleReq 修改角色请求体
type setUserRoleReq struct {
	Role string `json:"role" binding:"required"`
}

// SetUserRole godoc
// PUT /api/v1/admin/users/:id/role
// 修改用户角色（需 superadmin 权限，由路由层 RequireMinRole 保证）
// 可设置的目标角色：user / creator / moderator / admin
// （不允许通过此接口设置 superadmin，防止权限扩散）
func (h *RoleHandler) SetUserRole(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil || userID == 0 {
		response.FailBadRequest(c, "无效的用户 ID")
		return
	}

	var req setUserRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, err.Error())
		return
	}

	// 验证目标角色合法性（不允许通过此接口将用户提升为 superadmin）
	validRoles := map[string]bool{
		middleware.RoleUser:      true,
		middleware.RoleCreator:   true,
		middleware.RoleModerator: true,
		middleware.RoleAdmin:     true,
	}
	if !validRoles[req.Role] {
		response.FailBadRequest(c, "无效的角色，可选：user / creator / moderator / admin")
		return
	}

	if err := h.userSvc.SetUserRole(c.Request.Context(), userID, req.Role); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "role": req.Role})
}

// GetMyRole godoc
// GET /api/v1/me/role
// 获取当前已登录用户的角色（需 JWT）
func (h *RoleHandler) GetMyRole(c *gin.Context) {
	role := middleware.GetUserRoleStr(c)
	response.Success(c, gin.H{"role": role})
}
