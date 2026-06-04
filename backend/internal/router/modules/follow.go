// follow.go：关注/通知/Feed/WebSocket 模块路由。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/internal/ws"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterFollowRoutes 注册关注/通知/Feed/WebSocket 模块路由
func RegisterFollowRoutes(v1 *gin.RouterGroup, deps *Deps) {
	followSvc := service.NewFollowService(
		deps.FollowRepo,
		deps.UserRepo,
		deps.ProjectRepo,
		deps.TeamRepo,
		deps.DevLogRepo,
		deps.PostRepo,
		deps.DB,
	)
	deps.FollowSvc = followSvc
	followSvc.SetNotificationService(deps.NotifSvc)
	h := handler.NewFollowHandler(followSvc, deps.UserRepo)

	// 通知路由统一走 NotificationHandler（单一职责，通过 NotificationService 读写）
	nh := handler.NewNotificationHandler(deps.NotifSvc, deps.NotifPrefRepo)

	wsHandler := ws.NewHandler(deps.WSHub, deps.Log, deps.Cfg.Server.AllowOrigins)

	// 粉丝/关注列表（公开）
	v1.GET("/users/:id/followers", h.GetFollowers)
	v1.GET("/users/:id/following", h.GetFollowing)
	v1.GET("/projects/:id/followers", h.GetProjectFollowers)

	// 关注/取关操作（需 JWT）
	userFollowGroup := v1.Group("/users", middleware.JWTAuth())
	{
		userFollowGroup.POST("/:id/follow", h.FollowUser)
		userFollowGroup.DELETE("/:id/follow", h.UnfollowUser)
	}

	projectFollowGroup := v1.Group("/projects", middleware.JWTAuth())
	{
		projectFollowGroup.POST("/:id/follow", h.FollowProject)
		projectFollowGroup.DELETE("/:id/follow", h.UnfollowProject)
	}

	// 通知（需 JWT）—— 由 NotificationHandler 统一处理
	notificationGroup := v1.Group("/notifications", middleware.JWTAuth())
	{
		notificationGroup.GET("", nh.GetMyNotifications)
		notificationGroup.PATCH("/read-all", nh.MarkAllNotificationsRead) // PUT → PATCH
		notificationGroup.DELETE("", nh.ClearNotifications)
		notificationGroup.DELETE("/read", nh.DeleteReadNotifications)
		notificationGroup.GET("/unread-count", nh.GetUnreadNotificationCount)
		notificationGroup.GET("/categories", nh.GetNotificationCategories)
		notificationGroup.GET("/by-type", nh.GetNotificationsByType)
			notificationGroup.GET("/preferences", nh.GetNotificationPreferences)
			notificationGroup.PATCH("/preferences", nh.UpdateNotificationPreference)
		notificationGroup.PATCH("/:nid/read", nh.MarkNotificationRead) // PUT → PATCH
		notificationGroup.DELETE("/:nid", nh.DeleteNotification)
	}

	// WebSocket 实时推送（内部自行鉴权）
	v1.GET("/ws", wsHandler.ServeWS)


	// 话题关注（需 JWT）
	topicFollowGroup := v1.Group("/topics", middleware.JWTAuth())
	{
		topicFollowGroup.POST("/:id/follow", h.FollowTopic)
		topicFollowGroup.DELETE("/:id/follow", h.UnfollowTopic)
		topicFollowGroup.GET("/:id/follow", h.IsFollowingTopic)
	}

	// 我关注的话题列表（需 JWT）
	v1.GET("/me/followed-topics", middleware.JWTAuth(), h.GetMyFollowedTopics)
}
