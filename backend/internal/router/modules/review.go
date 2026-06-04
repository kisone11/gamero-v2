// review.go：玩家评测模块路由
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterReviewRoutes 注册玩家评测模块路由
func RegisterReviewRoutes(v1 *gin.RouterGroup, deps *Deps) {
	h := handler.NewReviewHandler(deps.ReviewSvc)

	// 公开接口（可选 JWT）
	v1.GET("/projects/:id/reviews", middleware.JWTAuthOptional(), h.ListReviews)
	v1.GET("/projects/:id/reviews/summary", h.GetRatingSummary)

	// 写操作（需 JWT）
	g := v1.Group("/projects", middleware.JWTAuth())
	{
		g.POST("/:id/reviews", h.CreateReview)
		g.PUT("/:id/reviews/:reviewId", h.UpdateReview)
		g.DELETE("/:id/reviews/:reviewId", h.DeleteReview)
		g.POST("/:id/reviews/:reviewId/like", h.LikeReview)
		g.DELETE("/:id/reviews/:reviewId/like", h.UnlikeReview)
		g.POST("/:id/reviews/:reviewId/reply", h.ReplyReview)
	}
}
