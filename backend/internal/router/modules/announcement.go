package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAnnouncementRoutes(v1 *gin.RouterGroup, deps *Deps) {
	h := handler.NewAnnouncementHandler(deps.DB)
	v1.GET("/announcements", h.ListPublic)

	admin := v1.Group("/admin/announcements", middleware.JWTAuth(), middleware.RequireRole(middleware.RoleAdmin, middleware.RoleSuperAdmin))
	{
		admin.GET("", h.AdminList)
		admin.POST("", h.AdminCreate)
		admin.PUT("/:id", h.AdminUpdate)
		admin.DELETE("/:id", h.AdminDelete)
	}
}
