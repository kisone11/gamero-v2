// discover.go：发现/搜索/推荐模块路由。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterDiscoverRoutes 注册发现/搜索/推荐模块路由
func RegisterDiscoverRoutes(v1 *gin.RouterGroup, deps *Deps) {
	discoverRepo := repository.NewDiscoverRepository(deps.DB)
	discoverSvc := service.NewDiscoverService(discoverRepo, deps.Stor, deps.Searcher)
	h := handler.NewDiscoverHandler(discoverSvc)

	// 统一搜索（无需登录）
	v1.GET("/search", h.UnifiedSearch)

	// 分类搜索（无需登录）
	searchGroup := v1.Group("/search")
	{
		searchGroup.GET("/projects", h.SearchProjects)
		searchGroup.GET("/users", h.SearchUsers)
		searchGroup.GET("/logs", h.SearchLogs)
		searchGroup.GET("/posts", h.SearchPosts)
	}

	// 个性化推荐（需 JWT）
	v1.Group("/discover", middleware.JWTAuth()).GET("/recommended-posts", h.GetRecommendedPosts)

	// 热门/分类浏览（无需登录）
	discoverPublicGroup := v1.Group("/discover")
	{
		// 前端调用的路径
		discoverPublicGroup.GET("/projects", h.GetHotProjects)         // 改自 hot-projects
		discoverPublicGroup.GET("/logs", h.GetHotLogsPaged)            // 改自 hot-logs-paged
		discoverPublicGroup.GET("/posts", h.GetHotPosts)               // 改自 hot-posts

		// ===== 兼容旧路由 =====
		discoverPublicGroup.GET("/hot-projects", h.GetHotProjects)
		discoverPublicGroup.GET("/hot-logs-paged", h.GetHotLogsPaged)
		discoverPublicGroup.GET("/hot-posts", h.GetHotPosts)
	}
}
