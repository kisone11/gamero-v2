// Package modules 提供路由模块化拆分的各模块实现。
// upload.go：媒体上传凭证模块路由（前端直传云存储方案）。
package modules

import (
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterUploadRoutes 注册媒体上传凭证路由。
//
// 该路由为前端直传云存储方案的第一步：
// 前端通过此接口获取预签名 PUT URL，然后直接将文件上传到云存储，
// 最后携带返回的 key 调用对应业务接口（如 PUT /me/avatar）完成业务绑定。
func RegisterUploadRoutes(v1 *gin.RouterGroup, deps *Deps) {
	// 构建 PostRepository（community 模块本地创建，这里也单独创建，无状态可复用）
	postRepo := repository.NewPostRepository(deps.DB)

	h := handler.NewUploadTokenHandler(
		deps.ProjectRepo,
		deps.DevLogRepo,
		postRepo,
	)

	// POST /api/v1/upload/token  → 需要 JWT 认证
	v1.POST("/upload/token", middleware.JWTAuth(), h.GetUploadToken)
}
