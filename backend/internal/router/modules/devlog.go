// devlog.go：开发日志模块路由。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterDevLogRoutes 注册开发日志模块路由
func RegisterDevLogRoutes(v1 *gin.RouterGroup, deps *Deps) {
	devLogCommentRepo := repository.NewDevLogCommentRepository(deps.DB)
	devLogLikeRepo := repository.NewDevLogLikeRepository(deps.DB)
	devLogCollectRepo := repository.NewDevLogCollectRepository(deps.DB)
	devLogCommentLikeRepo := repository.NewDevLogCommentLikeRepository(deps.DB)

	devLogSvc := service.NewDevLogService(
		deps.DevLogRepo,
		devLogCommentRepo,
		devLogLikeRepo,
		devLogCollectRepo,
		devLogCommentLikeRepo,
		deps.ProjectRepo,
		deps.UserRepo,
		deps.TeamRepo,
		deps.Stor,
		deps.Broker,
		deps.FollowRepo,
		deps.Searcher,
	)
	deps.DevLogSvc = devLogSvc

	// 注入成就服务
	if dn, ok := devLogSvc.(interface{ SetNotificationService(service.NotificationService) }); ok {
		dn.SetNotificationService(deps.NotifSvc)
	}

	h := handler.NewDevLogHandler(devLogSvc)

	// 公开部分（可选 JWT）
	v1.GET("/projects/:id/logs", middleware.JWTAuthOptional(), h.ListLogs)
	v1.GET("/logs/:logID", middleware.JWTAuthOptional(), h.GetLog)
	v1.GET("/projects/:id/releases", h.ListReleases)
	v1.GET("/logs/:logID/comments", middleware.JWTAuthOptional(), h.ListComments)

	// 创建日志（需 JWT，项目成员）
	v1.Group("/projects", middleware.JWTAuth()).POST("/:id/logs", h.CreateLog)

	// 日志需鉴权操作
	logGroup := v1.Group("/logs", middleware.JWTAuth())
	{
		logGroup.PATCH("/:logID", h.UpdateLog)              // PUT → PATCH
		logGroup.DELETE("/:logID", h.DeleteLog)
		logGroup.POST("/:logID/publish", h.PublishLog)
		logGroup.POST("/:logID/images", h.SaveLogImage)      // PUT → POST（对齐前端）
		logGroup.PUT("/:logID/videos", h.SaveLogVideo)       // 保留 PUT
		logGroup.DELETE("/:logID/videos", h.DeleteLogVideo)
		logGroup.DELETE("/:logID/images", h.DeleteLogImage)
		logGroup.POST("/:logID/like", h.LikeLog)
		logGroup.DELETE("/:logID/like", h.UnlikeLog)
		logGroup.POST("/:logID/collect", h.CollectLog)
		logGroup.DELETE("/:logID/collect", h.UncollectLog)
		logGroup.POST("/:logID/comments", h.CreateComment)
	}

	// 评论操作（需 JWT）
	commentGroup := v1.Group("/comments", middleware.JWTAuth())
	{
		commentGroup.DELETE("/:commentID", h.DeleteComment)
		commentGroup.POST("/:commentID/like", h.LikeComment)
		commentGroup.DELETE("/:commentID/like", h.UnlikeComment)
	}

	// 我的日志 & 收藏（需 JWT）
	meGroup := v1.Group("/me", middleware.JWTAuth())
	{
		meGroup.GET("/logs", h.GetMyLogs)
		meGroup.GET("/collections/logs", h.GetMyCollectedLogs)
	}
}
