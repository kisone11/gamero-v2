// Package modules 提供路由模块化拆分的各模块实现。
// moderation.go：内容审核回调路由（腾讯云 COS 审核回调）。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterModerationRoutes 注册内容审核回调路由。
// 此路由无需 JWT 认证（由腾讯云服务器调用，通过签名+IP白名单验证）。
func RegisterModerationRoutes(v1 *gin.RouterGroup, deps *Deps) {
	h := handler.NewModerationHandler(deps.ModerationSvc, deps.Cfg.Moderation.AllowedCallbackIPs)
	v1.POST("/callback/moderation", h.HandleCallback)
}
