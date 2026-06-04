// Package handler 提供 HTTP 处理器公共工具函数。
package handler

import (
	"fmt"
	"strconv"

	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/response"

	"github.com/gin-gonic/gin"
)

// parseUint64Param 从 gin.Context 路由参数中解析 uint64 类型 ID。
// 返回错误时调用方应立即 return。
func parseUint64Param(c *gin.Context, paramName string) (uint64, error) {
	paramStr := c.Param(paramName)
	id, err := strconv.ParseUint(paramStr, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid param %s: %q", paramName, paramStr)
	}
	return id, nil
}

// requireLogin 获取当前登录用户 ID。
// 未登录时自动响应 401 并返回 (0, false)，调用方应立即 return。
// 所有需要强制登录的 handler 应使用此函数替代直接调用 middleware.GetUserID。
func requireLogin(c *gin.Context) (uint64, bool) {
	userID, ok := middleware.GetUserID(c)
	if !ok || userID == 0 {
		response.FailCode(c, apperrors.CodeUnauthorized)
		return 0, false
	}
	return userID, true
}

// parseUint64Query 从 Query String 解析 uint64 类型参数。
// 若缺失或格式错误则返回 apperrors.CodeParamInvalid 错误。
func parseUint64Query(c *gin.Context, paramName string) (uint64, error) {
	value := c.Query(paramName)
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, apperrors.New(apperrors.CodeParamInvalid, "参数 "+paramName+" 格式无效")
	}
	return id, nil
}
