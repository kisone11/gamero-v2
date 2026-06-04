// stats.go：创作者数据统计模块路由。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterStatsRoutes 注册统计模块路由，同时初始化 deps.StatsSvc（供定时任务使用）
func RegisterStatsRoutes(v1 *gin.RouterGroup, deps *Deps) {
	statsRepo := repository.NewStatsRepository(deps.DB)
	statsSvc := service.NewStatsService(statsRepo, deps.UserRepo, deps.ProjectRepo)
	deps.StatsSvc = statsSvc
	h := handler.NewStatsHandler(statsSvc)

	// 我的数据大盘（需 JWT）— 同时注册两个路径供前后端兼容
	v1.GET("/stats/me", middleware.JWTAuth(), h.GetMyStats)
	v1.GET("/users/me/stats", middleware.JWTAuth(), h.GetMyStats)
	// 指定项目统计（需 JWT，项目 owner）
	v1.GET("/projects/:id/stats", middleware.JWTAuth(), h.GetProjectStats)
}
