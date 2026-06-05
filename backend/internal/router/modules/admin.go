// admin.go:管理后台路由(需 JWT + admin 权限)。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes 注册管理后台路由
func RegisterAdminRoutes(v1 *gin.RouterGroup, deps *Deps) {
	// 用 NewAdminServiceFull 以支持敏感词持久化 + 开发日志
	devLogRepo := repository.NewDevLogRepository(deps.DB)
	swRepo := repository.NewSensitiveWordRepository(deps.DB)
	adminSvc := service.NewAdminServiceFull(
		deps.AdminRepo, deps.TeamRepo, devLogRepo, swRepo, deps.Stor,
	)
	adminSvc.SetNotificationService(deps.NotifSvc)

	roleHandler := handler.NewRoleHandler(deps.UserSvc)
	adminHandler := handler.NewAdminHandler(adminSvc)

	admin := v1.Group("/admin")
	admin.Use(middleware.JWTAuth(), adminHandler.AdminAuthMiddleware())
	{
		// ----- 用户管理 -----
		admin.GET("/users", adminHandler.ListUsers)
		admin.GET("/users/:id", adminHandler.GetUser)
		admin.POST("/users/:id/ban", adminHandler.BanUser) // PUT → POST
		admin.POST("/users/batch-ban", adminHandler.BatchBanUsers)
		admin.DELETE("/users/:id/ban", adminHandler.UnbanUser)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.GET("/users/export", adminHandler.ExportUsers)
		admin.PATCH("/users/:id/role", // PUT → PATCH
			middleware.RequireMinRole(middleware.RoleSuperAdmin),
			roleHandler.SetUserRole,
		)

		// ----- 内容审核 -----
		admin.GET("/reports", adminHandler.ListReports)
		admin.PATCH("/reports/:id", adminHandler.HandleReport) // PUT → PATCH
		admin.GET("/posts", adminHandler.ListPosts)
		admin.DELETE("/posts/:id", adminHandler.AdminDeletePost)
		admin.PATCH("/posts/:id/hide", adminHandler.AdminDeletePost) // 新路由(复用 AdminDeletePost)
		admin.GET("/logs", adminHandler.ListLogs)
		admin.PATCH("/logs/:id/hide", adminHandler.AdminDeleteLog) // 管理员强制删除日志
		admin.DELETE("/comments/:id", adminHandler.AdminDeleteComment)
		admin.DELETE("/recruitments/:id", adminHandler.AdminCloseRecruitment)

		// ----- 话题管理 -----
		admin.POST("/topics", adminHandler.AdminCreateTopic)
		admin.GET("/topics", adminHandler.ListTopics)
		admin.PUT("/topics/:id", adminHandler.AdminUpdateTopic)
		admin.DELETE("/topics/:id", adminHandler.AdminDeleteTopic)
		admin.PUT("/topics/:id/icon", adminHandler.SaveTopicIcon)

		// ----- 统计数据 -----
		admin.GET("/dashboard", adminHandler.GetDashboardOverview)
		admin.GET("/stats", adminHandler.GetPlatformStats)
		admin.GET("/stats/daily", adminHandler.GetDailyStats)

		// ----- 审计日志 -----
		admin.GET("/audit-logs", adminHandler.GetAuditLogs)

		// ----- 项目管理 -----
		admin.GET("/projects", adminHandler.ListProjects)
		admin.PUT("/projects/:id/ban", adminHandler.BanProject)
		admin.DELETE("/projects/:id/ban", adminHandler.UnbanProject)
		admin.DELETE("/projects/:id", adminHandler.AdminDeleteProject)

		// ----- 敏感词管理 -----
		admin.GET("/sensitive-words", adminHandler.GetSensitiveWords)
		admin.POST("/sensitive-words", adminHandler.AddSensitiveWords)
		admin.DELETE("/sensitive-words/:word", adminHandler.DeleteSensitiveWord)
	}
}
