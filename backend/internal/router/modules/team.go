// Package modules
// team.go：组队/招募/申请/评价模块路由。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterTeamRoutes 注册组队/招募/申请/评价/人才邀请模块路由
func RegisterTeamRoutes(v1 *gin.RouterGroup, deps *Deps) {
	teamSvc := service.NewTeamService(deps.TeamRepo, deps.ProjectRepo, deps.UserRepo, deps.Stor)

	// 注入人才邀请仓储
	invitationRepo := repository.NewTalentInvitationRepository(deps.DB)
	if ts, ok := teamSvc.(interface {
		SetInvitationRepo(repository.TalentInvitationRepository)
	}); ok {
		ts.SetInvitationRepo(invitationRepo)
	}

	deps.TeamSvc = teamSvc

	// 注入通知服务（与成就服务无关，应无条件注入）
	teamSvc.SetNotificationService(deps.NotifSvc)

	h := handler.NewTeamHandler(teamSvc)

	// 公开浏览
	v1.GET("/recruitments", h.ListRecruitments)
	v1.GET("/recruitments/expiring-soon", h.GetExpiringSoonRecruitments)
	v1.GET("/recruitments/:id", h.GetRecruitment)
	v1.GET("/review-tags", h.GetPresetReviewTags)
	v1.GET("/talents", h.ListTalents)
	v1.GET("/users/:id/reviews", h.GetUserReviews)
	v1.GET("/users/:id/rating", h.GetUserRating)

	// 招募写操作（需 JWT）
	recruitmentGroup := v1.Group("/recruitments", middleware.JWTAuth())
	{
		recruitmentGroup.POST("", h.CreateRecruitment)
		recruitmentGroup.PUT("/:id", h.UpdateRecruitment)
		recruitmentGroup.PATCH("/:id/close", h.CloseRecruitment)   // POST → PATCH
		recruitmentGroup.POST("/:id/reopen", h.ReopenRecruitment)
		recruitmentGroup.DELETE("/:id", h.DeleteRecruitment)
		recruitmentGroup.POST("/:id/apply", h.ApplyRecruitment)
		recruitmentGroup.DELETE("/:id/apply", h.WithdrawApplication)
		recruitmentGroup.GET("/:id/applications", h.GetRecruitmentApplications)
	}

	// 申请操作（需 JWT）
	applicationGroup := v1.Group("/applications", middleware.JWTAuth())
	{
		applicationGroup.GET("/mine", h.GetMyApplications)
		applicationGroup.PATCH("/:appID/withdraw", h.WithdrawApplication) // 新路由
		applicationGroup.PATCH("/:appID/approve", h.ApproveApplication)   // POST → PATCH
		applicationGroup.PATCH("/:appID/reject", h.RejectApplication)     // POST → PATCH
	}

	// 评价相关（需 JWT）
	reviewGroup := v1.Group("/reviews", middleware.JWTAuth())
	{
		reviewGroup.POST("/:reviewID/supplement", h.AddReviewSupplement)
	}

	// 项目合作评价（需 JWT）
	// ─ 从 /projects/:id/collaboration-reviews 改为 /collab-reviews
	v1.POST("/collab-reviews", middleware.JWTAuth(), h.CreateReview)

	// ===== 人才合作偏好 =====
	// 前端调用 PATCH /users/me/availability
	v1.PATCH("/users/me/availability", middleware.JWTAuth(), h.SetAvailability)
	// 兼容旧路由
	v1.PUT("/me/availability", middleware.JWTAuth(), h.SetAvailability)

	// ===== 人才邀请 =====
	// 邀请操作（需 JWT）
	invGroup := v1.Group("/invitations", middleware.JWTAuth())
	{
		invGroup.POST("", h.InviteTalent)                           // 发送邀请（不需要 projectId 在路径中）
		invGroup.GET("/mine", h.GetMyInvitations)                   // 我收到的邀请
		invGroup.PATCH("/:id/accept", h.RespondInvitation)          // 接受邀请
		invGroup.PATCH("/:id/decline", h.RespondInvitation)         // 拒绝邀请
		invGroup.DELETE("/:id", h.WithdrawInvitation)               // 撤回邀请
	}

	// ===== 兼容旧路由：项目相关邀请 =====
	projectInvGroup := v1.Group("/projects", middleware.JWTAuth())
	{
		projectInvGroup.POST("/:id/invitations", h.InviteTalent)         // 发送邀请
		projectInvGroup.GET("/:id/invitations", h.GetProjectInvitations) // 查看项目发出的邀请
		projectInvGroup.POST("/:id/collaboration-reviews", h.CreateReview) // 兼容旧路由
	}
}
