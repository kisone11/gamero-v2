// Package middleware 提供 Gin HTTP 框架的中间件集合。
// 包含 JWT 认证、请求日志、CORS、限流和 Panic Recovery 等中间件。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gamero/gamero/pkg/auth"
	"github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	// UserIDKey 存储用户 ID 的 context key
	UserIDKey = "user_id"
	// UsernameKey 存储用户名的 context key
	UsernameKey = "username"
	// UserRoleKey 存储用户角色的 context key
	UserRoleKey = "user_role"
	// ClaimsKey 存储完整 JWT Claims 的 context key
	ClaimsKey = "jwt_claims"
)

// JWTAuth JWT 认证中间件
// 从请求头 Authorization: Bearer <token> 中提取并验证 Token
// 验证通过后，将用户信息存入 Gin Context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.FailCode(c, errors.CodeTokenMissing)
			c.Abort()
			return
		}

		// 支持 "Bearer <token>" 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.FailCode(c, errors.CodeTokenInvalid)
			c.Abort()
			return
		}

		tokenStr := parts[1]
		if tokenStr == "" {
			response.FailCode(c, errors.CodeTokenMissing)
			c.Abort()
			return
		}

		// 解析并验证 Token
		claims, err := auth.ParseAccessToken(tokenStr)
		if err != nil {
			response.Fail(c, err)
			c.Abort()
			return
		}

		// 检查 Token 是否已被吊销
		revoked, err := auth.IsTokenRevoked(c.Request.Context(), tokenStr)
		if err != nil {
			response.FailCode(c, errors.CodeInternalError)
			c.Abort()
			return
		}
		if revoked {
			response.FailCode(c, errors.CodeTokenRevoked)
			c.Abort()
			return
		}

		// 将用户信息存入 Context
		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Set(UserRoleKey, claims.Role)
		c.Set(ClaimsKey, claims)

		c.Next()
	}
}

// JWTAuthOptional 可选 JWT 认证中间件
// 如果有 Token 则验证并设置用户信息，没有 Token 则继续（不拦截）
func JWTAuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		tokenStr := parts[1]
		if tokenStr == "" {
			c.Next()
			return
		}

		claims, err := auth.ParseAccessToken(tokenStr)
		if err != nil {
			c.Next()
			return
		}

		revoked, err := auth.IsTokenRevoked(c.Request.Context(), tokenStr)
		if err != nil || revoked {
			c.Next()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Set(UserRoleKey, claims.Role)
		c.Set(ClaimsKey, claims)

		c.Next()
	}
}

// RequireRole 角色权限检查中间件
// 必须在 JWTAuth 之后使用
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(UserRoleKey)
		if !exists {
			response.FailCode(c, errors.CodeUnauthorized)
			c.Abort()
			return
		}

		userRole, ok := roleVal.(string)
		if !ok {
			response.FailCode(c, errors.CodeUnauthorized)
			c.Abort()
			return
		}

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, response.Response{
			Code:    errors.CodePermissionDenied,
			Message: errors.CodePermissionDenied.Message(),
		})
		c.Abort()
	}
}

// GetUserID 从 Gin Context 中获取当前用户 ID
func GetUserID(c *gin.Context) (uint64, bool) {
	val, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	id, ok := val.(uint64)
	return id, ok
}

// GetUsername 从 Gin Context 中获取当前用户名
func GetUsername(c *gin.Context) (string, bool) {
	val, exists := c.Get(UsernameKey)
	if !exists {
		return "", false
	}
	name, ok := val.(string)
	return name, ok
}

// GetUserRole 从 Gin Context 中获取当前用户角色
func GetUserRole(c *gin.Context) (string, bool) {
	val, exists := c.Get(UserRoleKey)
	if !exists {
		return "", false
	}
	role, ok := val.(string)
	return role, ok
}
