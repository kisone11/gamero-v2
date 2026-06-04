// Package service 提供内容审核业务逻辑层。
// moderation_service.go：处理腾讯云 COS 内容审核回调。
//
// 设计原则：
//  - 所有非致命错误使用 logger.Warn 记录，不阻断流程
//  - 审核未通过的资源依次执行：COS 删除 → 数据库引用清理 → 系统通知
package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/storage"
	"go.uber.org/zap"
)

// ===========================
// 回调请求结构体
// ===========================

// ModerationCallbackReq 腾讯云 COS 内容审核回调顶层结构。
type ModerationCallbackReq struct {
	EventName string                `json:"EventName"`
	Key       string                `json:"Key"`
	Records   []ModerationCallbackRec `json:"Records"`
}

// ModerationCallbackRec 审核回调中的单条记录。
type ModerationCallbackRec struct {
	EventName string               `json:"eventName"`
	Key       string               `json:"key"`
	Result    ModerationResultWrap `json:"result"`
}

// ModerationResultWrap 审核结果外层包装。
type ModerationResultWrap struct {
	Code int                  `json:"code"`
	Data ModerationResultData `json:"data"`
}

// ModerationResultData 审核结果数据层。
type ModerationResultData struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    ModerationResultDetail `json:"data"`
	Type    string                 `json:"type"`
}

// ModerationResultDetail 审核结果详情。
type ModerationResultDetail struct {
	ForbiddenStatus int `json:"forbidden_status"`
}

// ===========================
// 场景标签映射
// ===========================

// moderationSceneLabels 场景标签中文说明映射。
var moderationSceneLabels = map[string]string{
	"porn":     "色情内容",
	"terror":   "暴恐内容",
	"politics": "政治敏感内容",
	"ads":      "广告内容",
}

// ===========================
// 接口定义
// ===========================

// ModerationService 内容审核服务接口。
type ModerationService interface {
	// HandleCallback 处理腾讯云 COS 内容审核回调。
	// signature: HMAC-SHA256 签名字符串（Base64 编码）
	// body: 回调请求体原始字节
	HandleCallback(ctx context.Context, signature string, body []byte) error
}

// ===========================
// 实现
// ===========================

type moderationService struct {
	userRepo    repository.UserRepository
	projectRepo repository.ProjectRepository
	devLogRepo  repository.DevLogRepository
	postRepo    repository.PostRepository
	notifSvc    NotificationService
	secret      string
	allowedIPs  string
}

// NewModerationService 创建内容审核服务实例。
func NewModerationService(
	userRepo repository.UserRepository,
	projectRepo repository.ProjectRepository,
	devLogRepo repository.DevLogRepository,
	postRepo repository.PostRepository,
	notifSvc NotificationService,
	secret string,
	allowedIPs string,
) ModerationService {
	return &moderationService{
		userRepo:    userRepo,
		projectRepo: projectRepo,
		devLogRepo:  devLogRepo,
		postRepo:    postRepo,
		notifSvc:    notifSvc,
		secret:      secret,
		allowedIPs:  allowedIPs,
	}
}

// HandleCallback 处理内容审核回调：验证签名 → 解析请求 → 遍历记录 → 处理违规内容。
func (s *moderationService) HandleCallback(ctx context.Context, signature string, body []byte) error {
	// 1. 验证 HMAC-SHA256 签名
	if !s.verifySignature(signature, body) {
		logger.Warn("moderation callback signature verification failed")
		return nil // 静默拒绝，不暴露签名验证细节
	}

	// 2. 解析回调 JSON
	var req ModerationCallbackReq
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Warn("failed to unmarshal moderation callback", zap.Error(err))
		return nil
	}

	// 3. 遍历每条审核记录
	for _, rec := range req.Records {
		if s.isBlocked(rec) {
			key := rec.Key
			if key == "" {
				key = req.Key
			}
			scene := rec.Result.Data.Type
			s.handleBlocked(ctx, key, scene)
		}
	}

	return nil
}

// verifySignature 验证 HMAC-SHA256 签名。
// 腾讯云 CI 使用 Base64(secret) 作为 HMAC 密钥，对请求体进行 SHA256 签名后 Base64 输出。
func (s *moderationService) verifySignature(signature string, body []byte) bool {
	if signature == "" || s.secret == "" {
		return false
	}

	key, err := base64.StdEncoding.DecodeString(s.secret)
	if err != nil {
		logger.Warn("failed to decode moderation secret", zap.Error(err))
		return false
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// isBlocked 判断审核记录是否判定为违规（forbidden_status == 1）。
func (s *moderationService) isBlocked(rec ModerationCallbackRec) bool {
	return rec.Result.Data.Data.ForbiddenStatus == 1
}

// handleBlocked 处理违规内容：删除 COS 文件 → 清理数据库引用 → 通知内容所有者。
func (s *moderationService) handleBlocked(ctx context.Context, key, scene string) {
	logger.Info("handling moderated content",
		zap.String("key", key),
		zap.String("scene", scene),
	)

	// 1. 删除 COS 上的文件（失败仅记录日志，不阻断数据库清理）
	if err := storage.Delete(ctx, key); err != nil {
		logger.Warn("failed to delete moderated file from COS",
			zap.String("key", key),
			zap.Error(err),
		)
	}

	// 2. 根据 key 前缀路由到对应的类型处理
	switch {
	case strings.HasPrefix(key, "avatars/"):
		s.handleAvatarBlocked(ctx, key, scene)
	case strings.HasPrefix(key, "projects/"):
		if strings.Contains(key, "/cover/") {
			s.handleProjectCoverBlocked(ctx, key, scene)
		} else if strings.Contains(key, "/screenshots/") {
			s.handleProjectScreenshotBlocked(ctx, key, scene)
		}
	case strings.HasPrefix(key, "logs/"):
		if strings.Contains(key, "/images/") {
			s.handleDevLogImageBlocked(ctx, key, scene)
		} else if strings.Contains(key, "/videos/") {
			s.handleDevLogVideoBlocked(ctx, key, scene)
		}
	case strings.HasPrefix(key, "posts/"):
		if strings.Contains(key, "/images/") {
			s.handlePostImageBlocked(ctx, key, scene)
		} else if strings.Contains(key, "/videos/") {
			s.handlePostVideoBlocked(ctx, key, scene)
		}
	default:
		logger.Warn("unknown moderation content type, key not handled",
			zap.String("key", key),
		)
	}
}

// ===========================
// 类型处理器
// ===========================

// handleAvatarBlocked 处理头像违规：清空用户 avatar_key。
func (s *moderationService) handleAvatarBlocked(ctx context.Context, key, scene string) {
	userID := extractResourceID(key, "avatars")
	if userID == 0 {
		logger.Warn("unable to extract user ID from avatar key", zap.String("key", key))
		return
	}

	// 清空用户头像字段
	if err := s.userRepo.UpdateUser(ctx, userID, map[string]interface{}{
		"avatar_key": "",
	}); err != nil {
		logger.Warn("failed to clear avatar_key for moderated user",
			zap.Uint64("user_id", userID),
			zap.Error(err),
		)
		return
	}

	s.notifyOwner(ctx, userID, "头像", scene, key)
}

// handleProjectCoverBlocked 处理项目封面违规：清空项目 cover_key。
func (s *moderationService) handleProjectCoverBlocked(ctx context.Context, key, scene string) {
	projectID := extractResourceID(key, "projects")
	if projectID == 0 {
		logger.Warn("unable to extract project ID from cover key", zap.String("key", key))
		return
	}

	// 获取项目以确定 owner
	project, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		logger.Warn("failed to get project for moderated cover",
			zap.Uint64("project_id", projectID),
			zap.Error(err),
		)
		return
	}

	// 清空封面字段
	if err := s.projectRepo.UpdateProject(ctx, projectID, map[string]interface{}{
		"cover_key": "",
	}); err != nil {
		logger.Warn("failed to clear cover_key for moderated project",
			zap.Uint64("project_id", projectID),
			zap.Error(err),
		)
		return
	}

	s.notifyOwner(ctx, project.OwnerID, "图片", scene, key)
}

// handleProjectScreenshotBlocked 处理项目截图违规：从 screenshot_keys 中移除对应 key。
func (s *moderationService) handleProjectScreenshotBlocked(ctx context.Context, key, scene string) {
	projectID := extractResourceID(key, "projects")
	if projectID == 0 {
		logger.Warn("unable to extract project ID from screenshot key", zap.String("key", key))
		return
	}

	project, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		logger.Warn("failed to get project for moderated screenshot",
			zap.Uint64("project_id", projectID),
			zap.Error(err),
		)
		return
	}

	// 解析截图 key 列表，移除违规 key
	keys := parseStringSlice(project.ScreenshotKeys)
	newKeys := removeKeyFromSlice(keys, key)
	if len(keys) == len(newKeys) {
		// key 不在列表中，可能已被其他流程处理
		return
	}

	newKeysJSON, err := json.Marshal(newKeys)
	if err != nil {
		logger.Warn("failed to marshal screenshot keys",
			zap.Uint64("project_id", projectID),
			zap.Error(err),
		)
		return
	}

	if err := s.projectRepo.UpdateProject(ctx, projectID, map[string]interface{}{
		"screenshot_keys": string(newKeysJSON),
	}); err != nil {
		logger.Warn("failed to update screenshot_keys for moderated project",
			zap.Uint64("project_id", projectID),
			zap.Error(err),
		)
		return
	}

	s.notifyOwner(ctx, project.OwnerID, "图片", scene, key)
}

// handleDevLogImageBlocked 处理开发日志图片违规：从 images 列表中移除对应 key。
func (s *moderationService) handleDevLogImageBlocked(ctx context.Context, key, scene string) {
	logID := extractResourceID(key, "logs")
	if logID == 0 {
		logger.Warn("unable to extract log ID from image key", zap.String("key", key))
		return
	}

	devLog, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		logger.Warn("failed to get dev log for moderated image",
			zap.Uint64("log_id", logID),
			zap.Error(err),
		)
		return
	}

	// 解析图片 key 列表，移除违规 key
	images := parseStringSlice(devLog.Images)
	newImages := removeKeyFromSlice(images, key)
	if len(images) == len(newImages) {
		return
	}

	newImagesJSON, err := json.Marshal(newImages)
	if err != nil {
		logger.Warn("failed to marshal dev log images",
			zap.Uint64("log_id", logID),
			zap.Error(err),
		)
		return
	}

	if err := s.devLogRepo.UpdateLog(ctx, logID, map[string]interface{}{
		"images": string(newImagesJSON),
	}); err != nil {
		logger.Warn("failed to update images for moderated dev log",
			zap.Uint64("log_id", logID),
			zap.Error(err),
		)
		return
	}

	s.notifyOwner(ctx, devLog.AuthorID, "图片", scene, key)
}

// handleDevLogVideoBlocked 处理开发日志视频违规：从 videos 列表中移除对应 key。
func (s *moderationService) handleDevLogVideoBlocked(ctx context.Context, key, scene string) {
	logID := extractResourceID(key, "logs")
	if logID == 0 {
		logger.Warn("unable to extract log ID from video key", zap.String("key", key))
		return
	}

	devLog, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		logger.Warn("failed to get dev log for moderated video",
			zap.Uint64("log_id", logID),
			zap.Error(err),
		)
		return
	}

	// 解析视频 key 列表，移除违规 key
	videos := parseStringSlice(devLog.Videos)
	newVideos := removeKeyFromSlice(videos, key)
	if len(videos) == len(newVideos) {
		return
	}

	newVideosJSON, err := json.Marshal(newVideos)
	if err != nil {
		logger.Warn("failed to marshal dev log videos",
			zap.Uint64("log_id", logID),
			zap.Error(err),
		)
		return
	}

	if err := s.devLogRepo.UpdateLog(ctx, logID, map[string]interface{}{
		"videos": string(newVideosJSON),
	}); err != nil {
		logger.Warn("failed to update videos for moderated dev log",
			zap.Uint64("log_id", logID),
			zap.Error(err),
		)
		return
	}

	s.notifyOwner(ctx, devLog.AuthorID, "视频", scene, key)
}

// handlePostImageBlocked 处理帖子图片违规：从 images 列表中移除对应 key。
func (s *moderationService) handlePostImageBlocked(ctx context.Context, key, scene string) {
	postID := extractResourceID(key, "posts")
	if postID == 0 {
		logger.Warn("unable to extract post ID from image key", zap.String("key", key))
		return
	}

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		logger.Warn("failed to get post for moderated image",
			zap.Uint64("post_id", postID),
			zap.Error(err),
		)
		return
	}

	// 解析图片 key 列表，移除违规 key
	images := parseStringSlice(post.Images)
	newImages := removeKeyFromSlice(images, key)
	if len(images) == len(newImages) {
		return
	}

	newImagesJSON, err := json.Marshal(newImages)
	if err != nil {
		logger.Warn("failed to marshal post images",
			zap.Uint64("post_id", postID),
			zap.Error(err),
		)
		return
	}

	if err := s.postRepo.UpdatePost(ctx, postID, map[string]interface{}{
		"images": string(newImagesJSON),
	}); err != nil {
		logger.Warn("failed to update images for moderated post",
			zap.Uint64("post_id", postID),
			zap.Error(err),
		)
		return
	}

	s.notifyOwner(ctx, post.AuthorID, "图片", scene, key)
}

// handlePostVideoBlocked 处理帖子视频违规：从 videos 列表中移除对应 key。
func (s *moderationService) handlePostVideoBlocked(ctx context.Context, key, scene string) {
	postID := extractResourceID(key, "posts")
	if postID == 0 {
		logger.Warn("unable to extract post ID from video key", zap.String("key", key))
		return
	}

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		logger.Warn("failed to get post for moderated video",
			zap.Uint64("post_id", postID),
			zap.Error(err),
		)
		return
	}

	// 解析视频 key 列表，移除违规 key
	videos := parseStringSlice(post.Videos)
	newVideos := removeKeyFromSlice(videos, key)
	if len(videos) == len(newVideos) {
		return
	}

	newVideosJSON, err := json.Marshal(newVideos)
	if err != nil {
		logger.Warn("failed to marshal post videos",
			zap.Uint64("post_id", postID),
			zap.Error(err),
		)
		return
	}

	if err := s.postRepo.UpdatePost(ctx, postID, map[string]interface{}{
		"videos": string(newVideosJSON),
	}); err != nil {
		logger.Warn("failed to update videos for moderated post",
			zap.Uint64("post_id", postID),
			zap.Error(err),
		)
		return
	}

	s.notifyOwner(ctx, post.AuthorID, "视频", scene, key)
}

// ===========================
// 通知辅助
// ===========================

// notifyOwner 向内容所有者发送系统通知。
// mediaType 为中文描述（头像/图片/视频），scene 为英文场景标签（porn/terror/politics/ads）。
func (s *moderationService) notifyOwner(ctx context.Context, userID uint64, mediaType, scene, key string) {
	label := moderationSceneLabels[scene]
	if label == "" {
		label = scene
	}

	content := "您上传的" + mediaType + "因包含" + label + "已被移除"

	req := &SendNotificationReq{
		UserID:   userID,
		SenderID: 0,
		Type:     model.NotificationTypeSystem,
		Title:    "内容审核通知",
		Content:  content,
		Metadata: map[string]interface{}{
			"object_key": key,
			"scene":      scene,
		},
	}

	if err := s.notifSvc.Send(ctx, req); err != nil {
		logger.Warn("failed to send moderation notification",
			zap.Uint64("user_id", userID),
			zap.String("key", key),
			zap.Error(err),
		)
	}
}

// ===========================
// 工具函数
// ===========================

// extractResourceID 从对象存储 key 中提取资源 ID。
// key 格式: "{prefix}/{id}/..."，例如 "avatars/42/avatar.jpg" 返回 42。
// 解析失败时返回 0。
func extractResourceID(key, prefix string) uint64 {
	trimmed := strings.TrimPrefix(key, prefix+"/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return 0
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// removeKeyFromSlice 从字符串切片中移除指定的目标字符串，返回新切片。
func removeKeyFromSlice(keys []string, target string) []string {
	result := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != target {
			result = append(result, k)
		}
	}
	return result
}
