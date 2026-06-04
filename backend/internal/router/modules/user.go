// Package modules 提供路由模块化拆分的各模块实现。
// user.go:用户/认证/角色/拉黑模块路由。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes 注册用户/认证/角色模块路由
func RegisterUserRoutes(v1 *gin.RouterGroup, deps *Deps) {
	userSvc := service.NewUserService(deps.UserRepo, deps.TeamRepo)
	// 注入关注仓库(主页 is_following 字段)
	if deps.FollowRepo != nil {
		userSvc.SetFollowRepository(deps.FollowRepo)
	}
	// 注入开发日志仓库(日志分页查询)
	if deps.DevLogRepo != nil {
		userSvc.SetDevLogRepository(deps.DevLogRepo)
	}
	// 注入成就服务(资料完善成就触发)
	userSvc.SetNotificationService(deps.NotifSvc)

	userHandler := handler.NewUserHandler(userSvc)
	roleHandler := handler.NewRoleHandler(userSvc)

	// 认证模块(无需 JWT,前端 baseURL: /api/v1/auth)
	authGroup := v1.Group("/auth")
	{
		strictLimit := middleware.StrictRateLimiter(deps.Cfg.RateLimit)
		// authGroup.POST("/phone/code", strictLimit, userHandler.SendSMSCode)
		authGroup.POST("/email/code", strictLimit, userHandler.SendEmailCode)
		// authGroup.POST("/register/phone", strictLimit, userHandler.RegisterByPhone)
		authGroup.POST("/register/email", strictLimit, userHandler.RegisterByEmail)
		// authGroup.POST("/login/phone", strictLimit, userHandler.LoginByPhone)
		authGroup.POST("/login/email", strictLimit, userHandler.LoginByEmail)
		// authGroup.POST("/login/wechat", userHandler.LoginByWechat)
		// authGroup.POST("/login/github", userHandler.LoginByGithub)
		authGroup.POST("/refresh", strictLimit, userHandler.RefreshToken)
		authGroup.POST("/logout", middleware.JWTAuth(), userHandler.Logout)
		authGroup.POST("/password/change", middleware.JWTAuth(), userHandler.ChangePassword)
		authGroup.POST("/password/reset", strictLimit, userHandler.ResetPasswordByCode)
		authGroup.DELETE("/account", middleware.JWTAuth(), userHandler.DeleteAccount)
		// Deprecated routes (kept for backward compatibility)
		authGroup.POST("/forgot-password", strictLimit, userHandler.SendResetPasswordCode)
		authGroup.POST("/reset-password", userHandler.ResetPasswordByCode)
	}

	// 用户主页(公开,可选 JWT)
	v1.GET("/users/:id/logs", userHandler.GetUserLogs)
	v1.GET("/users/:id/endorsements", userHandler.GetUserEndorsements)
	v1.GET("/users/search", userHandler.SearchUsers)
	v1.GET("/users/:id", middleware.JWTAuthOptional(), userHandler.GetUserProfile)

	// 个人操作(需 JWT,前端 baseURL: /api/v1/users/me)
	meGroup := v1.Group("/users/me", middleware.JWTAuth())
	{
		// 基础信息
		meGroup.GET("", userHandler.GetMyProfile)            // GET /users/me
		meGroup.PATCH("/profile", userHandler.UpdateProfile) // PATCH /users/me/profile(前端用此路径)
		meGroup.PATCH("/avatar", userHandler.UploadAvatar)
		meGroup.PUT("/skills", userHandler.UpdateSkills)

		// 作品集
		meGroup.GET("/portfolio", userHandler.GetMyPortfolio)
		meGroup.POST("/portfolio", userHandler.CreatePortfolio)
		meGroup.PATCH("/portfolio/:id", userHandler.UpdatePortfolio)
		meGroup.DELETE("/portfolio/:id", userHandler.DeletePortfolio)

		// 其他
		meGroup.GET("/role", roleHandler.GetMyRole)
		// meGroup.GET("/stats", ...) // 注册在 stats.go 中(/users/me/stats)

		// meGroup.PATCH("/availability", ...) // 注册在 team.go 中(/users/me/availability)
	}

	// 兼容旧路由 /me/profile (已废弃,保留以防万一)
	v1.GET("/me/profile", middleware.JWTAuth(), userHandler.GetMyProfile)
}
