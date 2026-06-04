// feed.go：统一动态流模块路由。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterFeedRoutes 注册统一动态流模块路由
func RegisterFeedRoutes(v1 *gin.RouterGroup, deps *Deps) {
	h := handler.NewFeedHandler(deps.FeedSvc)

	// 统一动态流（需 JWT）
	v1.GET("/feed", middleware.JWTAuth(), h.GetFeed)
}
