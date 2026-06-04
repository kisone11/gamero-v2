// community.go：社区/帖子/评论/话题/举报模块路由（含 MQ subscriber 注册）。
package modules

import (
	"go.uber.org/zap"
	"github.com/gamero/gamero/pkg/logger"
	"context"
	"fmt"

	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/mq"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterCommunityRoutes 注册社区模块路由，同时注册 MQ subscriber 并设置 ProjectSvc broker
func RegisterCommunityRoutes(v1 *gin.RouterGroup, deps *Deps) {
	topicRepo := repository.NewTopicRepository(deps.DB)
	postRepo := repository.NewPostRepository(deps.DB)
	postCommentRepo := repository.NewPostCommentRepository(deps.DB)
	postLikeRepo := repository.NewPostLikeRepository(deps.DB)
	postCollectRepo := repository.NewPostCollectRepository(deps.DB)
	postCommentLikeRepo := repository.NewPostCommentLikeRepository(deps.DB)

	communitySvc := service.NewCommunityService(
		topicRepo,
		postRepo,
		postCommentRepo,
		postLikeRepo,
		postCollectRepo,
		postCommentLikeRepo,
		deps.ReportRepo,
		deps.UserRepo,
		deps.TeamRepo,
		deps.FollowRepo,
		deps.Stor,
		deps.Broker,
	)
	deps.CommunitySvc = communitySvc


	if cn, ok := communitySvc.(interface {
		SetNotificationService(service.NotificationService)
	}); ok {
		cn.SetNotificationService(deps.NotifSvc)
	}

	h := handler.NewCommunityHandler(communitySvc, deps.UserRepo)

	// ProjectSvc 注入 broker（写扩散 Feed 依赖 MQ）
	if deps.ProjectSvc != nil {
		deps.ProjectSvc.SetBroker(deps.Broker)
	}

	// 注册 MQ subscriber（@提及通知 + Feed 扩散）
	mq.RegisterSubscribers(deps.Broker, mq.SubscriberDeps{
		SendMentionNotifications: func(ctx context.Context, content string, authorID uint64, entityType string, entityID uint64) {
			usernames := service.ExtractMentions(content)
			for _, username := range usernames {
				user, err := deps.UserRepo.GetUserByUsername(ctx, username)
				if err != nil || user.ID == authorID {
					continue
				}
				if err := deps.TeamRepo.CreateNotification(ctx, &model.Notification{
					UserID:  user.ID,
					Type:    model.NotificationTypeMention,
					Title:   "有人在评论中提到了你",
					Content: "@" + username + " 在评论中提到了你",
				}); err != nil {
					logger.Warn("failed to create notification", zap.Error(err))
				}
			}
		},
		FanoutProjectUpdate: func(ctx context.Context, projectID uint64, updateType string, score float64) {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("panic in MQ subscriber", zap.Any("panic", r))
					}
				}()
				followerIDs, err := deps.FollowRepo.GetProjectFollowerIDsUpToLimit(ctx, projectID, cache.FanoutThreshold)
				if err != nil || len(followerIDs) == 0 {
					return
				}
				member := fmt.Sprintf("project:%d", projectID)
				rdb := cache.Get()
				if rdb == nil {
					return
				}
				pipe := rdb.Pipeline()
				for _, uid := range followerIDs {
					key := cache.UserFeedKey(uid)
					pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
					pipe.ZRemRangeByRank(ctx, key, 0, -(cache.FeedMaxLen + 1))
				}
				if _, err := pipe.Exec(ctx); err != nil {
					logger.Warn("failed to execute redis pipeline", zap.Error(err))
				}
			}()
		},
		FanoutUserPost: func(ctx context.Context, authorID uint64, postID uint64, score float64) {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("panic in MQ subscriber", zap.Any("panic", r))
					}
				}()
				followerIDs, err := deps.FollowRepo.GetUserFollowerIDsUpToLimit(ctx, authorID, cache.FanoutThreshold)
				if err != nil || len(followerIDs) == 0 {
					return
				}
				member := fmt.Sprintf("post:%d", postID)
				rdb := cache.Get()
				if rdb == nil {
					return
				}
				pipe := rdb.Pipeline()
				for _, uid := range followerIDs {
					key := cache.UserFeedKey(uid)
					pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
					pipe.ZRemRangeByRank(ctx, key, 0, -(cache.FeedMaxLen + 1))
				}
				if _, err := pipe.Exec(ctx); err != nil {
					logger.Warn("failed to execute redis pipeline", zap.Error(err))
				}
			}()
		},
	})

	// ===== 社区模块统一前缀 /community =====
	communityGroup := v1.Group("/community")

	// 话题（公开）
	communityGroup.GET("/topics", h.GetTopics)
	communityGroup.GET("/topics/:id", h.GetTopic)

	// 帖子（公开，可选 JWT）
	communityGroup.GET("/posts", middleware.JWTAuthOptional(), h.ListPosts)
	communityGroup.GET("/posts/:id", middleware.JWTAuthOptional(), h.GetPost)

	// 帖子（需 JWT）
	postGroup := communityGroup.Group("/posts", middleware.JWTAuth())
	{
		postGroup.POST("", h.CreatePost)
		postGroup.PATCH("/:id", h.UpdatePost)  // PUT → PATCH
		postGroup.DELETE("/:id", h.DeletePost)
		postGroup.PUT("/:id/images", h.SavePostImage)
			postGroup.DELETE("/:id/images", h.DeletePostImage)
		postGroup.PUT("/:id/videos", h.SavePostVideo)
		postGroup.DELETE("/:id/videos", h.DeletePostVideo)
		postGroup.POST("/:id/like", h.LikePost)
		postGroup.DELETE("/:id/like", h.UnlikePost)
		postGroup.POST("/:id/collect", h.CollectPost)
		postGroup.DELETE("/:id/collect", h.UncollectPost)
	}

	// 评论（公开）
	communityGroup.GET("/posts/:id/comments", middleware.JWTAuthOptional(), h.ListPostComments)
	// 评论（需 JWT）
	communityGroup.POST("/posts/:id/comments", middleware.JWTAuth(), h.CreatePostComment)
	communityGroup.DELETE("/posts/:id/comments/:commentID", middleware.JWTAuth(), h.DeletePostComment)

	// 评论点赞（需 JWT）
	communityGroup.POST("/posts/:id/comments/:commentID/like", middleware.JWTAuth(), h.LikePostComment)
	communityGroup.DELETE("/posts/:id/comments/:commentID/like", middleware.JWTAuth(), h.UnlikePostComment)

	// ===== 兼容旧路由（保留以防万一） =====
	// 用户帖子列表（不在 community 前缀下）
	v1.GET("/users/:id/posts", middleware.JWTAuthOptional(), h.GetUserPosts)

	// 举报（需 JWT，全局路由）
	v1.POST("/reports", middleware.JWTAuth(), h.CreateReport)

	// 我的帖子/收藏/举报（需 JWT）
	meGroup := v1.Group("/me", middleware.JWTAuth())
	{
		meGroup.GET("/posts", h.GetMyPosts)
		meGroup.GET("/collections/posts", h.GetMyCollectedPosts)
		meGroup.GET("/reports", h.GetMyReports)
	}
}
