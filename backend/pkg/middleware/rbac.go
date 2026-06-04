// Package middleware — RBAC 基于角色的访问控制中间件。
// 提供基于角色等级的权限检查，角色等级从低到高：
// guest(0) < user(1) < creator(2) < moderator(3) < admin(4) < superadmin(5)
//
// 使用方式：
//
//	router.Use(middleware.JWTAuth(), middleware.RequireMinRole(middleware.RoleAdmin))
package middleware

import (
	"github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

// 角色常量（与 model.UserRole 保持一致，以字符串形式在 context 中流转）
const (
	RoleGuest      = "guest"      // 未登录访客（无 JWT）
	RoleUser       = "user"       // 普通注册用户
	RoleCreator    = "creator"    // 认证创作者
	RoleModerator  = "moderator"  // 版主（可删帖，不可操作用户）
	RoleAdmin      = "admin"      // 管理员（封禁/解封用户等）
	RoleSuperAdmin = "superadmin" // 超级管理员（授权/降级 admin）
)

// roleLevel 角色等级映射（数字越大权限越高）
var roleLevel = map[string]int{
	RoleGuest:      0,
	RoleUser:       1,
	RoleCreator:    2,
	RoleModerator:  3,
	RoleAdmin:      4,
	RoleSuperAdmin: 5,
}

// RoleAtLeast 检查用户角色是否满足最低等级要求。
// 返回 true 表示 userRole 的等级 >= minRole 的等级。
// 未知角色（空字符串或非法值）视为 guest(0)。
func RoleAtLeast(userRole, minRole string) bool {
	uLevel, ok := roleLevel[userRole]
	if !ok {
		uLevel = roleLevel[RoleGuest]
	}
	mLevel, ok := roleLevel[minRole]
	if !ok {
		mLevel = roleLevel[RoleSuperAdmin] // 未知 minRole 视为最高权限，默认拒绝
	}
	return uLevel >= mLevel
}

// RequireMinRole 返回一个中间件，要求用户角色等级 >= minRole。
// 必须在 JWTAuth() 中间件之后使用（依赖已认证的用户信息注入 context）。
//
// 与 RequireRole（精确匹配 any-of）不同，RequireMinRole 使用等级比较：
// 例如 RequireMinRole(RoleAdmin) 允许 admin 和 superadmin 通过。
func RequireMinRole(minRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(UserRoleKey)
		if !exists {
			response.FailCode(c, errors.CodeUnauthorized)
			c.Abort()
			return
		}
		userRole, ok := roleVal.(string)
		if !ok || !RoleAtLeast(userRole, minRole) {
			response.FailCode(c, errors.CodePermissionDenied)
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetUserRoleStr 从 Gin Context 中获取当前用户角色字符串。
// 若未设置（未登录访客），返回 RoleGuest。
func GetUserRoleStr(c *gin.Context) string {
	val, exists := c.Get(UserRoleKey)
	if !exists {
		return RoleGuest
	}
	role, ok := val.(string)
	if !ok || role == "" {
		return RoleGuest
	}
	return role
}
