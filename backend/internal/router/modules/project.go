// project.go：项目模块路由。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterProjectRoutes 注册项目模块路由
func RegisterProjectRoutes(v1 *gin.RouterGroup, deps *Deps) {
	projectSvc := service.NewProjectService(
		deps.ProjectRepo, deps.UserRepo, deps.FollowRepo,
		deps.TeamRepo, deps.Stor, deps.Searcher,
	)
	deps.ProjectSvc = projectSvc // 为 community/follow 等模块共享

	// 注入成就服务
	if pn, ok := projectSvc.(interface {
		SetNotificationService(service.NotificationService)
	}); ok {
		pn.SetNotificationService(deps.NotifSvc)
	}
	// 注入开发日志仓库（用于项目详情的版本发布数统计）
	if deps.DevLogRepo != nil {
		if ps, ok := projectSvc.(interface {
			SetDevLogRepository(repository.DevLogRepository)
		}); ok {
			ps.SetDevLogRepository(deps.DevLogRepo)
		}
	}
	h := handler.NewProjectHandler(projectSvc)

	// 公开浏览（可选 JWT）
	v1.GET("/projects", h.ListProjects)
	v1.GET("/projects/slug/:slug", middleware.JWTAuthOptional(), h.GetProjectBySlug)
	v1.GET("/projects/:id", middleware.JWTAuthOptional(), h.GetProjectByID)
	v1.GET("/projects/:id/timeline", h.GetTimeline)
	v1.GET("/projects/:id/members", h.GetProjectMembers)

	// 用户项目列表（公开，可选 JWT）
	v1.GET("/users/:id/projects", middleware.JWTAuthOptional(), h.GetUserProjects)

	// 写操作（需 JWT）
	g := v1.Group("/projects", middleware.JWTAuth())
	{
		g.POST("", h.CreateProject)
		g.PATCH("/:id", h.UpdateProject) // PUT → PATCH
		g.DELETE("/:id", h.DeleteProject)
		g.PATCH("/:id/status", h.UpdateStatus) // PUT → PATCH

		// 图片（cover PUT，screenshots POST）
		g.PATCH("/:id/cover", h.SaveCover) // PUT → PATCH
		g.POST("/:id/screenshots", h.SaveScreenshot)
		// DeleteScreenshot 通过请求体 JSON 传 key（MinIO key 含斜线，不适合放 URL param）
		g.DELETE("/:id/screenshots", h.DeleteScreenshot)

		// 视频
		g.POST("/:id/videos", h.SaveVideo)
		// DeleteVideo 通过请求体 JSON 传 key
		g.DELETE("/:id/videos", h.DeleteVideo)

		// 成员
		g.POST("/:id/members", h.AddMember)
		g.PATCH("/:id/members/:memberID", h.UpdateMember) // PUT → PATCH
		// LeaveProject（me）必须在 RemoveMember（:memberID）之前注册，Gin 静态路由优先
		g.DELETE("/:id/members/me", h.LeaveProject) // 主动退出
		g.DELETE("/:id/members/:memberID", h.RemoveMember)

		// 收藏
		g.POST("/:id/collect", h.CollectProject)
		g.DELETE("/:id/collect", h.UncollectProject)

		// 转让所有者
		g.PATCH("/:id/transfer-owner", h.TransferOwner) // PUT → PATCH

		// 项目任务看板
		g.GET("/:id/tasks", h.ListTasks)
		g.POST("/:id/tasks", h.CreateTask)
		g.PATCH("/:id/tasks/:taskID", h.UpdateTask)
		g.DELETE("/:id/tasks/:taskID", h.DeleteTask)

	}

	// 我收藏的项目（需 JWT）
	meGroup := v1.Group("/me", middleware.JWTAuth())
	{
		meGroup.GET("/collections/projects", h.GetMyCollectedProjects)
	}
}
