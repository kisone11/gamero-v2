// Package handler 提供媒体上传凭证 HTTP 处理层。
// 本文件实现"前端直传云存储"方案的预签名 URL 分发接口。
//
// 上传流程：
//  1. 前端 POST /api/v1/upload/token 获取预签名 PUT URL 和对象 key
//  2. 前端使用预签名 URL 直接 PUT 文件到云存储
//  3. 前端携带 key 调用具体业务接口（保存头像/封面/截图等）
package handler

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gamero/gamero/internal/repository"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gamero/gamero/pkg/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// =====================
// 请求 / 响应 DTO
// =====================

// GetUploadTokenReq 获取上传凭证请求体
type GetUploadTokenReq struct {
	// Type 上传类型，决定权限校验逻辑和对象路径前缀。
	// 有效值：avatar | cover | screenshot | log-image | log-video |
	//         post-image | post-video
	Type string `json:"type" binding:"required"`

	// ResourceID 关联资源 ID（项目ID/日志ID/帖子ID 等）。
	// avatar 不需要此字段。
	ResourceID uint64 `json:"resource_id"`

	// Filename 原始文件名，用于推断扩展名（如 photo.jpg）。
	Filename string `json:"filename" binding:"required"`

	// ContentType 文件 MIME 类型（如 image/jpeg）。
	ContentType string `json:"content_type" binding:"required"`
}

// GetUploadTokenResp 获取上传凭证响应体
type GetUploadTokenResp struct {
	// UploadURL 预签名 PUT URL，前端直接 PUT 上传，有效期 15 分钟。
	UploadURL string `json:"upload_url"`

	// Key 对象存储 key，前端上传成功后提交给业务接口（如 /me/avatar）。
	Key string `json:"key"`

	// ExpiresAt Unix 时间戳（秒），预签名 URL 的过期时间。
	ExpiresAt int64 `json:"expires_at"`

	// AuditEnabled 是否开启了云端内容审核。
	// 图片和视频类公开内容（avatar/cover/screenshot/log-image/post-image/log-video/post-video）为 true，
	// 文件类暂不审核。
	AuditEnabled bool `json:"audit_enabled"`

	// AuditNote 审核说明（给前端展示用）。
	// 仅在 AuditEnabled=true 时返回。
	AuditNote string `json:"audit_note,omitempty"`
}

// =====================
// ContentType 白名单
// =====================

// imageContentTypes 图片类型白名单
var imageContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/gif":  {},
	"image/webp": {},
}

// videoContentTypes 视频类型白名单
var videoContentTypes = map[string]struct{}{
	"video/mp4":       {},
	"video/quicktime": {},
	"video/webm":      {},
}

// isImageCT 检查是否为允许的图片 MIME 类型
func isImageCT(ct string) bool {
	_, ok := imageContentTypes[ct]
	return ok
}

// isVideoCT 检查是否为允许的视频 MIME 类型
func isVideoCT(ct string) bool {
	_, ok := videoContentTypes[ct]
	return ok
}

// isFileCT 检查是否为聊天文件允许的 MIME 类型（图片 + 视频 + 文档）
func isFileCT(ct string) bool {
	if isImageCT(ct) || isVideoCT(ct) {
		return true
	}
	switch ct {
	case "application/pdf", "application/zip", "text/plain":
		return true
	}
	return false
}

// =====================
// Handler
// =====================

// UploadTokenHandler 上传凭证 Handler
type UploadTokenHandler struct {
	projectRepo repository.ProjectRepository // 用于验证项目 owner / 成员
	logRepo     repository.DevLogRepository  // 用于验证日志作者
	postRepo    repository.PostRepository    // 用于验证帖子作者
}

// NewUploadTokenHandler 创建 UploadTokenHandler 实例
func NewUploadTokenHandler(
	projectRepo repository.ProjectRepository,
	logRepo repository.DevLogRepository,
	postRepo repository.PostRepository,
) *UploadTokenHandler {
	return &UploadTokenHandler{
		projectRepo: projectRepo,
		logRepo:     logRepo,
		postRepo:    postRepo,
	}
}

// GetUploadToken 获取预签名上传 URL
// POST /api/v1/upload/token
//
// 根据 type 字段执行对应的权限校验，然后生成预签名 PUT URL 返回给前端。
// 前端拿到 upload_url 后直接 PUT 文件到云存储，上传完成后使用 key 调用业务接口。
func (h *UploadTokenHandler) GetUploadToken(c *gin.Context) {
	var req GetUploadTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailBadRequest(c, "参数错误："+err.Error())
		return
	}

	// 获取当前登录用户 ID（所有类型均需 JWT）
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		response.FailCode(c, apperrors.CodeTokenMissing)
		return
	}

	ctx := c.Request.Context()

	// 推断文件扩展名（从原始文件名获取）
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(req.Filename)), ".")
	if ext == "" {
		ext = "bin" // 无法推断时的兜底扩展名
	}

	var objectKey string

	switch req.Type {
	// ---- 头像（自己上传，无需 ResourceID）----
	case "avatar":
		if !isImageCT(req.ContentType) {
			response.FailBadRequest(c, "不支持的图片格式，仅允许 jpeg/png/gif/webp")
			return
		}
		objectKey = buildKey("avatars", userID, "", ext)

	// ---- 项目封面（需 owner）----
	case "cover":
		if req.ResourceID == 0 {
			response.FailBadRequest(c, "resource_id 为必填项（项目 ID）")
			return
		}
		if !isImageCT(req.ContentType) {
			response.FailBadRequest(c, "不支持的图片格式，仅允许 jpeg/png/gif/webp")
			return
		}
		isOwner, err := h.projectRepo.IsOwner(ctx, req.ResourceID, userID)
		if err != nil {
			response.FailInternal(c)
			return
		}
		if !isOwner {
			response.FailForbidden(c)
			return
		}
		objectKey = buildKey("projects", req.ResourceID, "cover", ext)

	// ---- 项目截图（需 owner）----
	case "screenshot":
		if req.ResourceID == 0 {
			response.FailBadRequest(c, "resource_id 为必填项（项目 ID）")
			return
		}
		if !isImageCT(req.ContentType) {
			response.FailBadRequest(c, "不支持的图片格式，仅允许 jpeg/png/gif/webp")
			return
		}
		isOwner, err := h.projectRepo.IsOwner(ctx, req.ResourceID, userID)
		if err != nil {
			response.FailInternal(c)
			return
		}
		if !isOwner {
			response.FailForbidden(c)
			return
		}
		objectKey = buildKey("projects", req.ResourceID, "screenshots", ext)

	// ---- 日志图片（需作者）----
	case "log-image":
		if req.ResourceID == 0 {
			response.FailBadRequest(c, "resource_id 为必填项（日志 ID）")
			return
		}
		if !isImageCT(req.ContentType) {
			response.FailBadRequest(c, "不支持的图片格式，仅允许 jpeg/png/gif/webp")
			return
		}
		log, err := h.logRepo.GetLogByID(ctx, req.ResourceID)
		if err != nil {
			response.Fail(c, err)
			return
		}
		if log.AuthorID != userID {
			response.FailForbidden(c)
			return
		}
		objectKey = buildKey("logs", req.ResourceID, "images", ext)

	// ---- 日志视频（需作者）----
	case "log-video":
		if req.ResourceID == 0 {
			response.FailBadRequest(c, "resource_id 为必填项（日志 ID）")
			return
		}
		if !isVideoCT(req.ContentType) {
			response.FailBadRequest(c, "不支持的视频格式，仅允许 mp4/quicktime/webm")
			return
		}
		log, err := h.logRepo.GetLogByID(ctx, req.ResourceID)
		if err != nil {
			response.Fail(c, err)
			return
		}
		if log.AuthorID != userID {
			response.FailForbidden(c)
			return
		}
		objectKey = buildKey("logs", req.ResourceID, "videos", ext)

	// ---- 帖子图片（需作者）----
	case "post-image":
		if req.ResourceID == 0 {
			response.FailBadRequest(c, "resource_id 为必填项（帖子 ID）")
			return
		}
		if !isImageCT(req.ContentType) {
			response.FailBadRequest(c, "不支持的图片格式，仅允许 jpeg/png/gif/webp")
			return
		}
		post, err := h.postRepo.GetPostByID(ctx, req.ResourceID)
		if err != nil {
			response.Fail(c, err)
			return
		}
		if post.AuthorID != userID {
			response.FailForbidden(c)
			return
		}
		objectKey = buildKey("posts", req.ResourceID, "images", ext)

	// ---- 帖子视频（需作者）----
	case "post-video":
		if req.ResourceID == 0 {
			response.FailBadRequest(c, "resource_id 为必填项（帖子 ID）")
			return
		}
		if !isVideoCT(req.ContentType) {
			response.FailBadRequest(c, "不支持的视频格式，仅允许 mp4/quicktime/webm")
			return
		}
		post, err := h.postRepo.GetPostByID(ctx, req.ResourceID)
		if err != nil {
			response.Fail(c, err)
			return
		}
		if post.AuthorID != userID {
			response.FailForbidden(c)
			return
		}
		objectKey = buildKey("posts", req.ResourceID, "videos", ext)

	// ---- 项目视频（需 owner）----
	case "project-video":
		if req.ResourceID == 0 {
			response.FailBadRequest(c, "resource_id 为必填项（项目 ID）")
			return
		}
		if !isVideoCT(req.ContentType) {
			response.FailBadRequest(c, "不支持的视频格式，仅允许 mp4/quicktime/webm")
			return
		}
		isOwner, err := h.projectRepo.IsOwner(ctx, req.ResourceID, userID)
		if err != nil {
			response.FailInternal(c)
			return
		}
		if !isOwner {
			response.FailForbidden(c)
			return
		}
		objectKey = buildKey("projects", req.ResourceID, "videos", ext)

	default:
		response.FailBadRequest(c, "不支持的上传类型："+req.Type)
		return
	}

	// 生成预签名 PUT URL，有效期 15 分钟
	const tokenExpiry = 15 * time.Minute
	uploadURL, err := storage.PresignedPutURL(ctx, objectKey, tokenExpiry)
	if err != nil {
		response.FailInternal(c)
		return
	}

	// 判断该上传类型是否需要云端内容审核
	// 图片和视频类公开内容开启审核，文件类暂不审核
	auditEnabledTypes := map[string]struct{}{
		"avatar":        {},
		"cover":         {},
		"screenshot":    {},
		"log-image":     {},
		"log-video":     {},
		"post-image":    {},
		"post-video":    {},
		"project-video": {},
	}
	_, auditEnabled := auditEnabledTypes[req.Type]
	var auditNote string
	if auditEnabled {
		auditNote = "图片上传后将自动进行内容审核，违规内容将被移除"
	}

	response.Success(c, &GetUploadTokenResp{
		UploadURL:    uploadURL,
		Key:          objectKey,
		ExpiresAt:    time.Now().Add(tokenExpiry).Unix(),
		AuditEnabled: auditEnabled,
		AuditNote:    auditNote,
	})
}

// =====================
// 内部工具函数
// =====================

// buildKey 生成对象存储 key。
// 格式示例：
//   - buildKey("avatars", 42, "", "jpg")       → avatars/42/{uuid}.jpg
//   - buildKey("projects", 7, "cover", "png")  → projects/7/cover/{uuid}.png
//   - buildKey("logs", 3, "images", "webp")    → logs/3/images/{uuid}.webp
func buildKey(prefix string, resourceID uint64, subdir, ext string) string {
	id := strconv.FormatUint(resourceID, 10)
	filename := uuid.New().String() + "." + ext
	if subdir == "" {
		return prefix + "/" + id + "/" + filename
	}
	return prefix + "/" + id + "/" + subdir + "/" + filename
}
