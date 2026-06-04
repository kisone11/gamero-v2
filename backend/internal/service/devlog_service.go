// Package service 提供开发日志系统的业务逻辑层。
// 本文件包含开发日志相关的所有业务逻辑：
//   - 日志发布、编辑、删除
//   - 日志可见性控制（public / members_only）
//   - 日志点赞、收藏
//   - 评论发布、删除、点赞
//   - 日志图片上传（云存储）
//   - 发布时自动写入项目时间轴
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/mq"
	"github.com/gamero/gamero/pkg/search"
	"github.com/gamero/gamero/pkg/storage"
	"go.uber.org/zap"
)

// 开发日志媒体相关常量
const (
	// devLogMaxImages 每条日志最多6张图片
	devLogMaxImages = 6
	// devLogMaxVideos 每条日志最多3个视频
	devLogMaxVideos = 3
	// devLogPresignExpiry 预签名 URL 有效期 24 小时
	devLogPresignExpiry = 24 * time.Hour
)

// ===========================
// 请求/响应数据结构
// ===========================

// CreateDevLogReq 创建开发日志请求
type CreateDevLogReq struct {
	Title      string                   `json:"title"`      // 日志标题，最多128字符
	Content    string                   `json:"content"`    // 日志正文（Markdown），最多50000字符
	LogType    model.DevLogType         `json:"log_type"`   // 日志类型：log / release
	Version    string                   `json:"version"`    // 版本号，log_type=release 时必填
	Images     []string                 `json:"images"`     // 对象存储 key 列表，最多6张
	Videos     []string                 `json:"videos"`     // 对象存储 key 列表，最多3个
	Visibility model.DevLogVisibility   `json:"visibility"` // 可见范围：public / members_only
	Status     model.DevLogStatus       `json:"status"`     // 状态：draft / published
}

// UpdateDevLogReq 更新开发日志请求
type UpdateDevLogReq struct {
	Title       *string                  `json:"title"`        // 日志标题
	Content     *string                  `json:"content"`      // 日志正文
	LogType     *model.DevLogType        `json:"log_type"`     // 日志类型
	Version     *string                  `json:"version"`      // 版本号
	Images      []string                 `json:"images"`       // 对象存储 key 列表（覆盖更新）
	Videos      []string                 `json:"videos"`       // 对象存储 key 列表（覆盖更新）
	DownloadURL *string                  `json:"download_url"` // 下载链接（release 类型使用）
	Visibility  *model.DevLogVisibility  `json:"visibility"`   // 可见范围
	Status      *model.DevLogStatus      `json:"status"`       // 状态
}

// DevLogDetail 日志详情响应（包含作者信息和互动状态）
type DevLogDetail struct {
	*model.DevLog
	AuthorInfo       *DevLogAuthorInfo `json:"author"`               // 作者信息（嵌套对象）
	AuthorNickname   string            `json:"author_nickname"`      // 作者昵称（兼容前端顶层访问）
	AuthorUsername   string            `json:"author_username"`      // 作者用户名（兼容前端顶层访问）
	AuthorAvatarURL  string            `json:"author_avatar_url"`    // 作者头像 URL（兼容前端顶层访问）
	ImageURLs        []string          `json:"image_urls"`           // 图片公开 URL 列表
		ImageKeys        []string          `json:"image_keys"`           // 对象存储 key 列表
	VideoURLs        []string          `json:"video_urls"`           // 视频公开 URL 列表
	CoverURL         string            `json:"cover_url"`            // 封面图 URL（取第一张图片作为封面）
		VideoKeys        []string          `json:"video_keys"`           // 对象存储 key 列表
	IsLiked          bool              `json:"is_liked"`             // 当前用户是否已点赞
	IsCollected      bool              `json:"is_collected"`         // 当前用户是否已收藏
	ProjectName      string            `json:"project_name,omitempty"`      // 所属项目名称
	ProjectSlug      string            `json:"project_slug,omitempty"`      // 所属项目 slug
	ProjectCoverURL  string            `json:"project_cover_url,omitempty"` // 所属项目封面 URL
	ScreenshotURLs   []string          `json:"screenshot_urls,omitempty"`   // 所属项目截图 URL 列表
}

// DevLogAuthorInfo 日志作者基本信息
type DevLogAuthorInfo struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarKey string `json:"-"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// CommentWithReplies 带子评论的评论结构（用于树形返回）
type CommentWithReplies struct {
	*model.DevLogComment
	AuthorInfo  *DevLogAuthorInfo     `json:"author"`      // 评论作者信息
	IsLiked     bool                  `json:"is_liked"`    // 当前用户是否已点赞
	Replies     []*CommentWithReplies `json:"replies"`     // 子评论列表
}

// CreateCommentReq 发表评论请求
type CreateCommentReq struct {
	Content   string `json:"content"`     // 评论内容，最多1000字符
	ReplyToID uint64 `json:"reply_to_id"` // 回复的评论 ID，一级评论传 0
}

// UploadLogImageResult 图片上传结果
type UploadLogImageResult struct {
	Key string `json:"key"` // 对象存储 key
	URL string `json:"url"` // 预签名访问 URL
}

// UploadLogVideoResult 视频上传结果
type UploadLogVideoResult struct {
	Key      string `json:"key"`               // 对象存储 key
	URL      string `json:"url"`               // 预签名访问 URL
	MimeType string `json:"mime_type"`         // 视频 MIME 类型
	Size     int64  `json:"size"`              // 文件字节数
}

// ===========================
// Service 接口定义
// ===========================

// DevLogService 开发日志业务接口
type DevLogService interface {
	// CreateLog 创建开发日志（验证作者是项目成员，发布时写时间轴）
	CreateLog(ctx context.Context, authorID, projectID uint64, req *CreateDevLogReq) (*model.DevLog, error)
	// UpdateLog 更新日志（只有作者可编辑，已发布日志不可改为草稿）
	UpdateLog(ctx context.Context, authorID, logID uint64, req *UpdateDevLogReq) (*model.DevLog, error)
	// PublishLog 将草稿发布（写时间轴）
	PublishLog(ctx context.Context, authorID, logID uint64) (*model.DevLog, error)
	// DeleteLog 软删除日志（只有作者可删除）
	DeleteLog(ctx context.Context, authorID, logID uint64) error
	// GetLog 获取日志详情（含作者信息、互动状态，浏览数 +1）
	GetLog(ctx context.Context, logID, viewerID uint64) (*DevLogDetail, error)
	// ListLogs 分页查询项目日志列表（根据 viewerID 过滤 members_only）
	ListLogs(ctx context.Context, projectID uint64, params *ListLogsParams) ([]*DevLogDetail, int64, error)
	// ListReleases 查询版本历史列表
	ListReleases(ctx context.Context, projectID uint64, page, pageSize int) ([]*model.DevLog, int64, error)
	// LikeLog 点赞日志（已点赞则幂等）
	LikeLog(ctx context.Context, userID, logID uint64) error
	// UnlikeLog 取消点赞
	UnlikeLog(ctx context.Context, userID, logID uint64) error
	// CollectLog 收藏日志（已收藏则幂等）
	CollectLog(ctx context.Context, userID, logID uint64) error
	// UncollectLog 取消收藏
	UncollectLog(ctx context.Context, userID, logID uint64) error
	// CreateComment 发表评论
	CreateComment(ctx context.Context, authorID, logID uint64, req *CreateCommentReq) (*model.DevLogComment, error)
	// DeleteComment 删除评论（作者或日志作者均可删）
	DeleteComment(ctx context.Context, operatorID, commentID uint64) error
	// ListComments 分页查询评论（树形结构）
	ListComments(ctx context.Context, logID uint64, viewerID uint64, page, pageSize int, sortBy string) ([]*CommentWithReplies, int64, error)
	// LikeComment 评论点赞
	LikeComment(ctx context.Context, userID, commentID uint64) error
	// UnlikeComment 取消评论点赞
	UnlikeComment(ctx context.Context, userID, commentID uint64) error
	// SaveLogImage 保存日志图片 key（前端已直传后提交）
	SaveLogImage(ctx context.Context, authorID, logID uint64, key string) (*UploadLogImageResult, error)
	// SaveLogVideo 保存日志视频 key（前端已直传后提交）
	SaveLogVideo(ctx context.Context, authorID, logID uint64, key string) (*UploadLogVideoResult, error)
	// DeleteLogImage 删除日志中的单个图片（通过 对象存储 key）
	DeleteLogImage(ctx context.Context, authorID, logID uint64, key string) error
	// DeleteLogVideo 删除日志中的单个视频（通过 对象存储 key）
	DeleteLogVideo(ctx context.Context, authorID, logID uint64, key string) error
	// GetMyCollectedLogs 获取当前用户收藏的日志列表（分页）
	GetMyCollectedLogs(ctx context.Context, userID uint64, page, pageSize int) ([]*DevLogDetail, int64, error)
	// GetMyLogs 获取当前用户发布的日志列表（分页）
	GetMyLogs(ctx context.Context, userID uint64, page, pageSize int) ([]*model.DevLog, int64, error)
}

// ListLogsParams 日志列表查询参数
type ListLogsParams struct {
	LogType    model.DevLogType       // 按类型筛选
	Status     model.DevLogStatus     // 按状态筛选
	ViewerID   uint64                 // 查看者 ID（0表示未登录）
	Page       int
	PageSize   int
}

// ===========================
// Service 实现
// ===========================

// devLogService 开发日志 service 实现

type devLogService struct {
	devLogRepo        repository.DevLogRepository
	commentRepo       repository.DevLogCommentRepository
	likeRepo          repository.DevLogLikeRepository
	collectRepo       repository.DevLogCollectRepository
	commentLikeRepo   repository.DevLogCommentLikeRepository
	projectRepo       repository.ProjectRepository
	userRepo          repository.UserRepository
	teamRepo          repository.TeamRepository
	*NotificationClient
	followRepo        repository.FollowRepository // 用于写扩散时查询粉丝列表
	stor              *storage.Client
	broker            mq.Broker
	searcher          search.Searcher
	logger            *zap.Logger
}

// NewDevLogService 创建 DevLogService 实例
func NewDevLogService(
	devLogRepo repository.DevLogRepository,
	commentRepo repository.DevLogCommentRepository,
	likeRepo repository.DevLogLikeRepository,
	collectRepo repository.DevLogCollectRepository,
	commentLikeRepo repository.DevLogCommentLikeRepository,
	projectRepo repository.ProjectRepository,
	userRepo repository.UserRepository,
	teamRepo repository.TeamRepository,
	stor *storage.Client,
	broker mq.Broker,
	followRepo repository.FollowRepository,
	searcher search.Searcher,
) DevLogService {
	svc := &devLogService{
		devLogRepo:      devLogRepo,
		commentRepo:     commentRepo,
		likeRepo:        likeRepo,
		collectRepo:     collectRepo,
		commentLikeRepo: commentLikeRepo,
		projectRepo:     projectRepo,
		userRepo:        userRepo,
		teamRepo:        teamRepo,
		followRepo:      followRepo,
		stor:            stor,
		broker:          broker,
		NotificationClient: &NotificationClient{},
		searcher:        searcher,
		logger:          logger.Get(),
	}
	return svc
}



// sendNotification 写 DB 通知并实时 WS 推送（失败不阻断主流程）
func (s *devLogService) sendNotification(ctx context.Context, n *model.Notification) {
	s.SendModel(ctx, n)
}

// ===========================
// 类型转换辅助函数
// ===========================

// DevLogTypeFromString 将字符串安全转换为 DevLogType，无效值返回空
func DevLogTypeFromString(s string) model.DevLogType {
	t := model.DevLogType(s)
	if t == model.DevLogTypeLog || t == model.DevLogTypeRelease {
		return t
	}
	return ""
}

// DevLogStatusFromString 将字符串安全转换为 DevLogStatus，无效值返回空
func DevLogStatusFromString(s string) model.DevLogStatus {
	st := model.DevLogStatus(s)
	if st == model.DevLogStatusDraft || st == model.DevLogStatusPublished {
		return st
	}
	return ""
}

// ===========================
// 验证辅助方法
// ===========================

// validateCreateLogReq 校验创建日志请求参数
func validateCreateLogReq(req *CreateDevLogReq) error {
	if strings.TrimSpace(req.Title) == "" {
		return apperrors.New(apperrors.CodeParamMissing, "日志标题不能为空")
	}
	if len([]rune(req.Title)) > 128 {
		return apperrors.New(apperrors.CodeParamTooLong, "日志标题最多128字符")
	}
	if len([]rune(req.Content)) > 50000 {
		return apperrors.New(apperrors.CodeParamTooLong, "日志正文最多50000字符")
	}
	if req.LogType != model.DevLogTypeLog && req.LogType != model.DevLogTypeRelease {
		return apperrors.New(apperrors.CodeParamInvalid, "日志类型无效，仅支持 log / release")
	}
	if req.LogType == model.DevLogTypeRelease && strings.TrimSpace(req.Version) == "" {
		return apperrors.CodeError(apperrors.CodeDevLogVersionRequired)
	}
	if len([]rune(req.Version)) > 32 {
		return apperrors.New(apperrors.CodeParamTooLong, "版本号最多32字符")
	}
	if req.Visibility != model.DevLogVisibilityPublic && req.Visibility != model.DevLogVisibilityMembersOnly {
		// 默认公开
		req.Visibility = model.DevLogVisibilityPublic
	}
	if req.Status != model.DevLogStatusDraft && req.Status != model.DevLogStatusPublished {
		// 默认草稿
		req.Status = model.DevLogStatusDraft
	}
	if len(req.Images) > devLogMaxImages {
		return apperrors.CodeError(apperrors.CodeDevLogImageLimit)
	}
	if len(req.Videos) > devLogMaxVideos {
		return apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("视频最多%d个", devLogMaxVideos))
	}
	return nil
}

// getAuthorInfo 从 User 获取作者信息 DTO

func (s *devLogService) presignURL(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	url, err := s.stor.PresignedGetURL(ctx, key, 24*time.Hour)
	if err != nil {
		return s.stor.GetPublicURL(key)
	}
	return url
}
func getAuthorInfoFromUser(ctx context.Context, u *model.User) *DevLogAuthorInfo {
	if u == nil {
		return nil
	}
	avatarURL := ""
	if u.AvatarKey != "" {
		avatarURL = presignURL(ctx, u.AvatarKey)
	}
	return &DevLogAuthorInfo{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		AvatarKey: u.AvatarKey,
		AvatarURL: avatarURL,
	}
}

// writePublishTimeline 发布日志时写入项目时间轴
func (s *devLogService) writePublishTimeline(ctx context.Context, log *model.DevLog) error {
	title := fmt.Sprintf("发布了开发日志：%s", log.Title)
	content := ""
	if log.LogType == model.DevLogTypeRelease {
		title = fmt.Sprintf("发布了版本更新：%s %s", log.Version, log.Title)
		content = log.Version
	}
	event := &model.ProjectTimeline{
		ProjectID: log.ProjectID,
		Type:      model.TimelineEventLogPublished,
		Title:     title,
		Content:   content,
		Metadata:  json.RawMessage(fmt.Sprintf(`{"log_id":%d,"log_type":"%s"}`, log.ID, log.LogType)),
	}
	return s.projectRepo.CreateTimelineEvent(ctx, event)
}

// ===========================
// CreateLog
// ===========================

// CreateLog 创建开发日志
func (s *devLogService) CreateLog(ctx context.Context, authorID, projectID uint64, req *CreateDevLogReq) (*model.DevLog, error) {
	// 验证请求参数
	if err := validateCreateLogReq(req); err != nil {
		return nil, err
	}

	// 敏感词检测
	if filter.Contains(req.Title) || filter.Contains(req.Content) {
		return nil, apperrors.New(apperrors.CodeContentViolation, "日志内容包含敏感词，请修改后重试")
	}

	// 验证项目存在
	_, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 验证作者是项目成员
	isMember, err := s.projectRepo.IsMember(ctx, projectID, authorID)
	if err != nil {
		return nil, fmt.Errorf("验证项目成员失败: %w", err)
	}
	if !isMember {
		return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	// 序列化图片列表
	imagesJSON := "[]"
	if len(req.Images) > 0 {
		b, err := json.Marshal(req.Images)
		if err != nil {
			return nil, fmt.Errorf("序列化图片列表失败: %w", err)
		}
		imagesJSON = string(b)
	}

	// 序列化视频列表
	videosJSON := "[]"
	if len(req.Videos) > 0 {
		b, err := json.Marshal(req.Videos)
		if err != nil {
			return nil, fmt.Errorf("序列化视频列表失败: %w", err)
		}
		videosJSON = string(b)
	}

	log := &model.DevLog{
		ProjectID:  projectID,
		AuthorID:   authorID,
		Title:      strings.TrimSpace(req.Title),
		Content:    req.Content,
		LogType:    req.LogType,
		Version:    strings.TrimSpace(req.Version),
		Images:     imagesJSON,
		Videos:     videosJSON,
		Visibility: req.Visibility,
		Status:     req.Status,
	}

	if err := s.devLogRepo.CreateLog(ctx, log); err != nil {
		return nil, err
	}

	// 发布时写入时间轴
	if log.Status == model.DevLogStatusPublished {
		if err := s.writePublishTimeline(ctx, log); err != nil {
			// 时间轴写入失败不影响主流程
			logger.Warn("failed to write publish timeline", zap.Uint64("log_id", log.ID), zap.Error(err))
		}
		// 异步写扩散：将日志推入粉丝的 Feed ZSet
		s.asyncFanoutLog(log)
	}

	return log, nil
}

// ===========================
// UpdateLog
// ===========================

// UpdateLog 更新开发日志
func (s *devLogService) UpdateLog(ctx context.Context, authorID, logID uint64, req *UpdateDevLogReq) (*model.DevLog, error) {
	// 查询日志
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return nil, err
	}

	// 验证只有作者可编辑
	if log.AuthorID != authorID {
		return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, apperrors.New(apperrors.CodeParamMissing, "日志标题不能为空")
		}
		if len([]rune(title)) > 128 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "日志标题最多128字符")
		}
		if filter.Contains(title) {
			return nil, apperrors.New(apperrors.CodeContentViolation, "标题包含敏感词，请修改后重试")
		}
		updates["title"] = title
	}

	if req.Content != nil {
		content := *req.Content
		if len([]rune(content)) > 50000 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "日志正文最多50000字符")
		}
		if filter.Contains(content) {
			return nil, apperrors.New(apperrors.CodeContentViolation, "正文包含敏感词，请修改后重试")
		}
		updates["content"] = content
	}

	if req.LogType != nil {
		if *req.LogType != model.DevLogTypeLog && *req.LogType != model.DevLogTypeRelease {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "日志类型无效，仅支持 log / release")
		}
		updates["log_type"] = *req.LogType
	}

	if req.Version != nil {
		if len([]rune(*req.Version)) > 32 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "版本号最多32字符")
		}
		updates["version"] = strings.TrimSpace(*req.Version)
	}

	// 验证版本号（更新后的逻辑类型判断）
	finalLogType := log.LogType
	if req.LogType != nil {
		finalLogType = *req.LogType
	}
	finalVersion := log.Version
	if req.Version != nil {
		finalVersion = strings.TrimSpace(*req.Version)
	}
	if finalLogType == model.DevLogTypeRelease && strings.TrimSpace(finalVersion) == "" {
		return nil, apperrors.CodeError(apperrors.CodeDevLogVersionRequired)
	}

	if req.Images != nil {
		if len(req.Images) > devLogMaxImages {
			return nil, apperrors.CodeError(apperrors.CodeDevLogImageLimit)
		}
		b, err := json.Marshal(req.Images)
		if err != nil {
			return nil, fmt.Errorf("序列化图片列表失败: %w", err)
		}
		updates["images"] = string(b)
	}

	if req.DownloadURL != nil {
		if len(*req.DownloadURL) > 512 {
			return nil, apperrors.New(apperrors.CodeParamTooLong, "下载链接最多512字符")
		}
		updates["download_url"] = *req.DownloadURL
	}

	if req.Videos != nil {
		if len(req.Videos) > devLogMaxVideos {
			return nil, apperrors.New(apperrors.CodeParamInvalid, fmt.Sprintf("视频最多%d个", devLogMaxVideos))
		}
		b, err := json.Marshal(req.Videos)
		if err != nil {
			return nil, fmt.Errorf("序列化视频列表失败: %w", err)
		}
		updates["videos"] = string(b)
	}

	if req.Visibility != nil {
		if *req.Visibility != model.DevLogVisibilityPublic && *req.Visibility != model.DevLogVisibilityMembersOnly {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "可见范围无效，仅支持 public / members_only")
		}
		updates["visibility"] = *req.Visibility
	}

	if req.Status != nil {
		// 已发布日志不可改为草稿
		if log.Status == model.DevLogStatusPublished && *req.Status == model.DevLogStatusDraft {
			return nil, apperrors.CodeError(apperrors.CodeDevLogCannotUnpublish)
		}
		if *req.Status != model.DevLogStatusDraft && *req.Status != model.DevLogStatusPublished {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "状态无效，仅支持 draft / published")
		}
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		return log, nil
	}

	if err := s.devLogRepo.UpdateLog(ctx, logID, updates); err != nil {
		return nil, err
	}

	// 如果是从草稿改为发布，写时间轴
	if req.Status != nil && *req.Status == model.DevLogStatusPublished && log.Status == model.DevLogStatusDraft {
		// 应用更新到 log 对象后写时间轴
		if req.Title != nil {
			log.Title = *req.Title
		}
		if req.LogType != nil {
			log.LogType = *req.LogType
		}
		if req.Version != nil {
			log.Version = *req.Version
		}
		if err := s.writePublishTimeline(ctx, log); err != nil {
			s.logger.Warn("devlog: write publish timeline failed", zap.Uint64("log_id", logID), zap.Error(err))
		}
	}

	// 重新查询返回最新数据
	updated, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return nil, err
	}

	// 已发布的日志在更新后同步 ES 索引
	if updated.Status == model.DevLogStatusPublished {
		go s.indexLog(context.Background(), updated)
	}

	return updated, nil
}

// ===========================
// PublishLog
// ===========================

// PublishLog 将草稿发布
func (s *devLogService) PublishLog(ctx context.Context, authorID, logID uint64) (*model.DevLog, error) {
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return nil, err
	}

	// 只有作者可操作
	if log.AuthorID != authorID {
		return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	// 已发布则幂等返回
	if log.Status == model.DevLogStatusPublished {
		return log, nil
	}

	// 版本更新类型必须有版本号
	if log.LogType == model.DevLogTypeRelease && strings.TrimSpace(log.Version) == "" {
		return nil, apperrors.CodeError(apperrors.CodeDevLogVersionRequired)
	}

	if err := s.devLogRepo.UpdateLog(ctx, logID, map[string]interface{}{
		"status": model.DevLogStatusPublished,
	}); err != nil {
		return nil, err
	}

	log.Status = model.DevLogStatusPublished

	// 写入时间轴
	if err := s.writePublishTimeline(ctx, log); err != nil {
		logger.Warn("failed to write publish timeline", zap.Uint64("log_id", log.ID), zap.Error(err))
	}

	// 异步写扩散：将日志推入粉丝的 Feed ZSet
	s.asyncFanoutLog(log)
	// 异步更新 ES 索引
	go s.indexLog(context.Background(), log)

	return log, nil
}

// ===========================
// DeleteLog
// ===========================

// DeleteLog 软删除日志（只有作者可删除）
func (s *devLogService) DeleteLog(ctx context.Context, authorID, logID uint64) error {
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return err
	}

	if log.AuthorID != authorID {
		return apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	if err := s.devLogRepo.DeleteLog(ctx, logID); err != nil {
		return err
	}

	// 异步从 ES 删除索引（已发布日志才有索引）
	if log.Status == model.DevLogStatusPublished {
		go func() {
			if delErr := s.searcher.DeleteLog(context.Background(), logID); delErr != nil {
				s.logger.Warn("search: failed to delete log from ES", zap.Uint64("log_id", logID), zap.Error(delErr))
			}
		}()
	}
	return nil
}

// ===========================
// GetLog
// ===========================

// GetLog 获取日志详情（浏览数 +1，含作者信息和互动状态）
func (s *devLogService) GetLog(ctx context.Context, logID, viewerID uint64) (*DevLogDetail, error) {
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return nil, err
	}

	// 可见性控制：members_only 只有项目成员可查看
	if log.Visibility == model.DevLogVisibilityMembersOnly {
		if viewerID == 0 {
			return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
		}
		isMember, err := s.projectRepo.IsMember(ctx, log.ProjectID, viewerID)
		if err != nil {
			return nil, fmt.Errorf("验证项目成员失败: %w", err)
		}
		if !isMember {
			return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
		}
	}

	// 仅已发布的日志才对非成员可见（草稿只有作者可查看）
	if log.Status == model.DevLogStatusDraft && log.AuthorID != viewerID {
		return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	// UV 去重：同一访客当天只计一次浏览
	go func() {
		hllKey := cache.LogViewHLLKey(logID, time.Now().Format("2006-01-02"))
		member := fmt.Sprintf("%d", viewerID)
		isNew, _ := cache.PFAddAndCheckNew(context.Background(), hllKey, member, cache.TTLViewHLL)
		if isNew {
			if err := s.devLogRepo.IncrViewCount(context.Background(), logID); err != nil {
				logger.Warn("failed to update counts", zap.Error(err))
			}
		}
	}()

	// 查询作者信息
	author, err := s.userRepo.GetUserByID(ctx, log.AuthorID)
	if err != nil {
		author = nil // 作者信息获取失败不影响主数据返回
	}

	detail := &DevLogDetail{
		DevLog:     log,
		AuthorInfo: getAuthorInfoFromUser(ctx, author),
	}
	s.populateDevLogDetail(ctx, detail)

	// 查询当前用户互动状态
	if viewerID > 0 {
		liked, _ := s.likeRepo.HasLiked(ctx, logID, viewerID)
		collected, _ := s.collectRepo.HasCollected(ctx, logID, viewerID)
		detail.IsLiked = liked
		detail.IsCollected = collected
	}

	return detail, nil
}

// ===========================
// ListLogs
// ===========================

// ListLogs 分页查询项目日志列表
func (s *devLogService) ListLogs(ctx context.Context, projectID uint64, params *ListLogsParams) ([]*DevLogDetail, int64, error) {
	// 验证项目存在
	_, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}

	// 计算是否为成员（用于过滤 members_only 日志）
	isMember := false
	if params.ViewerID > 0 {
		isMember, _ = s.projectRepo.IsMember(ctx, projectID, params.ViewerID)
	}

	// 构造 repository 查询参数
	repoParams := &repository.ListDevLogsParams{
		LogType:  params.LogType,
		Status:   params.Status,
		Offset:   (params.Page - 1) * params.PageSize,
		Limit:    params.PageSize,
	}

	// 非成员只能看公开日志
	if !isMember {
		repoParams.Visibility = model.DevLogVisibilityPublic
	}

	// 非成员只能看已发布日志
	if !isMember && params.Status == "" {
		repoParams.Status = model.DevLogStatusPublished
	}

	logs, total, err := s.devLogRepo.ListLogsByProjectID(ctx, projectID, repoParams)
	if err != nil {
		return nil, 0, err
	}

	// 批量获取作者信息，避免 for 内单条查询 N+1
	authorIDs := make([]uint64, 0, len(logs))
	for _, log := range logs {
		authorIDs = append(authorIDs, log.AuthorID)
	}
	authors, _ := s.userRepo.GetUsersByIDs(ctx, authorIDs)
	authorMap := make(map[uint64]*model.User, len(authors))
	for _, u := range authors {
		authorMap[u.ID] = u
	}

	// 批量构造 detail
	details := make([]*DevLogDetail, 0, len(logs))
	for _, log := range logs {
		detail := &DevLogDetail{
			DevLog:     log,
			AuthorInfo: getAuthorInfoFromUser(ctx, authorMap[log.AuthorID]),
		}
	s.populateDevLogDetail(ctx, detail)
		if params.ViewerID > 0 {
			liked, _ := s.likeRepo.HasLiked(ctx, log.ID, params.ViewerID)
			collected, _ := s.collectRepo.HasCollected(ctx, log.ID, params.ViewerID)
			detail.IsLiked = liked
			detail.IsCollected = collected
		}
		details = append(details, detail)
	}

	return details, total, nil
}

// ===========================
// ListReleases
// ===========================

// ListReleases 版本历史列表
func (s *devLogService) ListReleases(ctx context.Context, projectID uint64, page, pageSize int) ([]*model.DevLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	_, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}

	return s.devLogRepo.ListReleasesByProjectID(ctx, projectID, offset, pageSize)
}

// ===========================
// 点赞/取消点赞
// ===========================

// LikeLog 点赞日志（已点赞则幂等）
func (s *devLogService) LikeLog(ctx context.Context, userID, logID uint64) error {
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return err
	}

	// 只能对已发布且公开（或成员可见且用户是成员）的日志点赞
	if log.Status != model.DevLogStatusPublished {
		return apperrors.CodeError(apperrors.CodeDevLogNotPublished)
	}
	if log.Visibility == model.DevLogVisibilityMembersOnly {
		isMember, err := s.projectRepo.IsMember(ctx, log.ProjectID, userID)
		if err != nil {
			return fmt.Errorf("验证成员失败: %w", err)
		}
		if !isMember {
			return apperrors.CodeError(apperrors.CodeDevLogForbidden)
		}
	}

	// 已点赞则幂等
	hasLiked, err := s.likeRepo.HasLiked(ctx, logID, userID)
	if err != nil {
		return fmt.Errorf("查询点赞状态失败: %w", err)
	}
	if hasLiked {
		return nil
	}

	if err := s.likeRepo.CreateLike(ctx, &model.DevLogLike{
		LogID:  logID,
		UserID: userID,
	}); err != nil {
		return err
	}

	// 同步更新冗余计数
	count, err := s.likeRepo.CountLikes(ctx, logID)
	if err == nil {
		if err := s.devLogRepo.UpdateCounts(ctx, logID, map[string]interface{}{"like_count": count}); err != nil {
			logger.Warn("failed to update counts", zap.Error(err))
		}
	}

	// 异步通知日志作者（不通知自己）
	if log.AuthorID != userID {
		go func() {
			liker, err := s.userRepo.GetUserByID(ctx, userID)
			if err != nil {
				return
			}
			name := liker.Nickname
			if name == "" {
				name = liker.Username
			}
			title := log.Title
			if len([]rune(title)) > 20 {
				title = string([]rune(title)[:20]) + "..."
			}
			s.sendNotification(ctx, &model.Notification{
				UserID:   log.AuthorID,
				Type:     model.NotificationTypeLogLiked,
				Title:    "有人赞了你的开发日志",
				Content:  fmt.Sprintf("%s 赞了你的日志「%s」", name, title),
				Metadata: fmt.Sprintf(`{"log_id":%d,"liker_id":%d}`, logID, userID),
			})
		}()
	}

	return nil
}

// UnlikeLog 取消点赞
func (s *devLogService) UnlikeLog(ctx context.Context, userID, logID uint64) error {
	_, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return err
	}

	if err := s.likeRepo.DeleteLike(ctx, logID, userID); err != nil {
		return err
	}

	// 同步更新冗余计数
	count, err := s.likeRepo.CountLikes(ctx, logID)
	if err == nil {
		if err := s.devLogRepo.UpdateCounts(ctx, logID, map[string]interface{}{"like_count": count}); err != nil {
			logger.Warn("failed to update counts", zap.Error(err))
		}
	}

	return nil
}

// ===========================
// 收藏/取消收藏
// ===========================

// CollectLog 收藏日志（已收藏则幂等）
func (s *devLogService) CollectLog(ctx context.Context, userID, logID uint64) error {
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return err
	}

	if log.Status != model.DevLogStatusPublished {
		return apperrors.CodeError(apperrors.CodeDevLogNotPublished)
	}
	if log.Visibility == model.DevLogVisibilityMembersOnly {
		isMember, err := s.projectRepo.IsMember(ctx, log.ProjectID, userID)
		if err != nil {
			return fmt.Errorf("验证成员失败: %w", err)
		}
		if !isMember {
			return apperrors.CodeError(apperrors.CodeDevLogForbidden)
		}
	}

	// 已收藏则幂等
	hasCollected, err := s.collectRepo.HasCollected(ctx, logID, userID)
	if err != nil {
		return fmt.Errorf("查询收藏状态失败: %w", err)
	}
	if hasCollected {
		return nil
	}

	if err := s.collectRepo.CreateCollect(ctx, &model.DevLogCollect{
		LogID:  logID,
		UserID: userID,
	}); err != nil {
		return err
	}

	// 同步更新收藏数冗余字段（原子加1）
	curLog, err2 := s.devLogRepo.GetLogByID(ctx, logID)
	if err2 == nil {
		if err := s.devLogRepo.UpdateCounts(ctx, logID, map[string]interface{}{"collect_count": curLog.CollectCount + 1}); err != nil {
			logger.Warn("failed to update counts", zap.Error(err))
		}
	}

	return nil
}

// UncollectLog 取消收藏
func (s *devLogService) UncollectLog(ctx context.Context, userID, logID uint64) error {
	_, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return err
	}

	if err := s.collectRepo.DeleteCollect(ctx, logID, userID); err != nil {
		return err
	}

	// 同步更新冗余计数（减一，最小为0）
	curLog, err2 := s.devLogRepo.GetLogByID(ctx, logID)
	if err2 == nil && curLog.CollectCount > 0 {
		if err := s.devLogRepo.UpdateCounts(ctx, logID, map[string]interface{}{"collect_count": curLog.CollectCount - 1}); err != nil {
			logger.Warn("failed to update counts", zap.Error(err))
		}
	}

	return nil
}

// ===========================
// 评论
// ===========================

// CreateComment 发表评论
func (s *devLogService) CreateComment(ctx context.Context, authorID, logID uint64, req *CreateCommentReq) (*model.DevLogComment, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, apperrors.New(apperrors.CodeParamMissing, "评论内容不能为空")
	}
	if len([]rune(content)) > 1000 {
		return nil, apperrors.New(apperrors.CodeParamTooLong, "评论内容最多1000字符")
	}

	// 查询日志
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return nil, err
	}

	// 日志必须已发布
	if log.Status != model.DevLogStatusPublished {
		return nil, apperrors.CodeError(apperrors.CodeDevLogNotPublished)
	}

	// 可见性控制
	if log.Visibility == model.DevLogVisibilityMembersOnly {
		isMember, err := s.projectRepo.IsMember(ctx, log.ProjectID, authorID)
		if err != nil {
			return nil, fmt.Errorf("验证成员失败: %w", err)
		}
		if !isMember {
			return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
		}
	}

	// 如果是子评论，验证父评论存在且属于同一日志
	if req.ReplyToID > 0 {
		parent, err := s.commentRepo.GetCommentByID(ctx, req.ReplyToID)
		if err != nil {
			return nil, apperrors.New(apperrors.CodeCommentNotFound, "被回复的评论不存在")
		}
		if parent.LogID != logID {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "被回复的评论不属于此日志")
		}
		// 只支持二级，不能回复子评论（即 reply_to_id 必须是一级评论）
		if parent.ReplyToID != 0 {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "仅支持二级评论，不能回复子评论")
		}
	}

	comment := &model.DevLogComment{
		LogID:     logID,
		AuthorID:  authorID,
		Content:   content,
		ReplyToID: req.ReplyToID,
	}

	if err := s.commentRepo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	// 同步更新日志评论数冗余字段
	count, err := s.commentRepo.CountCommentsByLogID(ctx, logID)
	if err == nil {
		if err := s.devLogRepo.UpdateCounts(ctx, logID, map[string]interface{}{"comment_count": count}); err != nil {
			logger.Warn("failed to update counts", zap.Error(err))
		}
	}

	// 异步通过消息队列发布评论创建事件（触发 @提及通知等），不阻塞主流程
	if s.broker != nil {
		event, err := mq.NewEvent(mq.EventCommentCreated, mq.CommentCreatedPayload{
			CommentID: comment.ID,
			PostID:    0,
			LogID:     logID,
			AuthorID:  comment.AuthorID,
			Content:   comment.Content,
		})
		if err == nil {
			if err := s.broker.Publish(ctx, event); err != nil {
				logger.Warn("failed to publish event", zap.Error(err))
			}
		}
	}

	// 异步通知日志作者（自己评论自己的日志不通知）
	if log.AuthorID != authorID {
		go func() {
			bgCtx := context.Background()
			preview := []rune(content)
			if len(preview) > 30 {
				preview = preview[:30]
			}
			n := &model.Notification{
				UserID:   log.AuthorID,
				Type:     model.NotificationTypeLogCommented,
				SenderID: authorID,
				Title:    "你的开发日志收到了新评论",
				Content:  string(preview),
				Metadata: fmt.Sprintf(`{"log_id":%d,"comment_id":%d}`, logID, comment.ID),
			}
			s.sendNotification(bgCtx, n)
		}()
	}

	return comment, nil
}

// sendMentionNotifications 解析评论内容中的 @username，向被提及用户发送通知（异步调用）
func (s *devLogService) sendMentionNotifications(ctx context.Context, authorID uint64, content string, commentID uint64) {
	usernames := extractMentions(content)
	for _, username := range usernames {
		user, err := s.userRepo.GetUserByUsername(ctx, username)
		if err != nil || user.ID == authorID {
			continue
		}
		s.Send(ctx, &SendNotificationReq{
			UserID: user.ID,
			Type: model.NotificationTypeMention,
			Title: "有人在评论中提到了你",
			Content: fmt.Sprintf("@%s 在评论中提到了你", username),
			Metadata: map[string]interface{}{"comment_id": commentID},
		})
	}
}

// DeleteComment 删除评论（作者或日志作者均可删）
func (s *devLogService) DeleteComment(ctx context.Context, operatorID, commentID uint64) error {
	comment, err := s.commentRepo.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	// 查询日志（获取日志作者）
	log, err := s.devLogRepo.GetLogByID(ctx, comment.LogID)
	if err != nil {
		return err
	}

	// 只有评论者或日志作者可删除
	if comment.AuthorID != operatorID && log.AuthorID != operatorID {
		return apperrors.CodeError(apperrors.CodeCommentForbidden)
	}

	if err := s.commentRepo.DeleteComment(ctx, commentID); err != nil {
		return err
	}

	// 同步更新日志评论数
	count, err := s.commentRepo.CountCommentsByLogID(ctx, comment.LogID)
	if err == nil {
		if err := s.devLogRepo.UpdateCounts(ctx, comment.LogID, map[string]interface{}{"comment_count": count}); err != nil {
			logger.Warn("failed to update counts", zap.Error(err))
		}
	}

	return nil
}

// ListComments 分页查询评论（树形结构）
func (s *devLogService) ListComments(ctx context.Context, logID uint64, viewerID uint64, page, pageSize int, sortBy string) ([]*CommentWithReplies, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询一级评论
	topComments, total, err := s.commentRepo.ListCommentsByLogID(ctx, logID, offset, pageSize, sortBy)
	if err != nil {
		return nil, 0, err
	}

	if len(topComments) == 0 {
		return []*CommentWithReplies{}, total, nil
	}

	// 收集一级评论 ID
	parentIDs := make([]uint64, 0, len(topComments))
	for _, c := range topComments {
		parentIDs = append(parentIDs, c.ID)
	}

	// 批量查询子评论
	childComments, err := s.commentRepo.ListChildCommentsByParentIDs(ctx, logID, parentIDs)
	if err != nil {
		return nil, 0, err
	}

	// 批量收集评论作者ID + 评论 ID，一次查询（避免 N+1）
	allAuthorIDs := make([]uint64, 0, len(topComments)+len(childComments))
	allCommentIDs := make([]uint64, 0, len(topComments)+len(childComments))
	for _, c := range topComments {
		allAuthorIDs = append(allAuthorIDs, c.AuthorID)
		allCommentIDs = append(allCommentIDs, c.ID)
	}
	for _, c := range childComments {
		allAuthorIDs = append(allAuthorIDs, c.AuthorID)
		allCommentIDs = append(allCommentIDs, c.ID)
	}

	// 批量查用户
	userMap := make(map[uint64]*model.User)
	if users, err := s.userRepo.GetUsersByIDs(ctx, allAuthorIDs); err == nil {
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	// 批量查点赞状态
	likedMap := make(map[uint64]bool)
	if viewerID > 0 {
		likedMap, _ = s.commentLikeRepo.GetLikedCommentIDs(ctx, viewerID, allCommentIDs)
	}

	// 构建父评论 ID → 子评论列表的映射
	childMap := make(map[uint64][]*CommentWithReplies)
	for _, child := range childComments {
		childMap[child.ReplyToID] = append(childMap[child.ReplyToID], &CommentWithReplies{
			DevLogComment: child,
			AuthorInfo:    getAuthorInfoFromUser(ctx, userMap[child.AuthorID]),
			IsLiked:       likedMap[child.ID],
			Replies:       []*CommentWithReplies{},
		})
	}

	// 组装树形结构
	result := make([]*CommentWithReplies, 0, len(topComments))
	for _, c := range topComments {
		replies := childMap[c.ID]
		if replies == nil {
			replies = []*CommentWithReplies{}
		}
		result = append(result, &CommentWithReplies{
			DevLogComment: c,
			AuthorInfo:    getAuthorInfoFromUser(ctx, userMap[c.AuthorID]),
			IsLiked:       likedMap[c.ID],
			Replies:       replies,
		})
	}

	return result, total, nil
}

// ===========================
// 评论点赞
// ===========================

// LikeComment 评论点赞（已点赞则幂等）
func (s *devLogService) LikeComment(ctx context.Context, userID, commentID uint64) error {
	_, err := s.commentRepo.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	// 已点赞则幂等
	hasLiked, err := s.commentLikeRepo.HasCommentLiked(ctx, commentID, userID)
	if err != nil {
		return fmt.Errorf("查询评论点赞状态失败: %w", err)
	}
	if hasLiked {
		return nil
	}

	if err := s.commentLikeRepo.CreateCommentLike(ctx, &model.DevLogCommentLike{
		CommentID: commentID,
		UserID:    userID,
	}); err != nil {
		return err
	}

	// 同步更新评论点赞数冗余字段（+1）
	if err := s.commentRepo.IncrCommentLikeCount(ctx, commentID, 1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}

	return nil
}

// UnlikeComment 取消评论点赞
func (s *devLogService) UnlikeComment(ctx context.Context, userID, commentID uint64) error {
	_, err := s.commentRepo.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	if err := s.commentLikeRepo.DeleteCommentLike(ctx, commentID, userID); err != nil {
		return err
	}

	// 同步更新评论点赞数冗余字段（-1）
	if err := s.commentRepo.IncrCommentLikeCount(ctx, commentID, -1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}

	return nil
}

// ===========================
// 媒体 key 保存
// ===========================

// SaveLogImage 保存日志图片 key（前端已通过预签名 URL 直传后提交）
func (s *devLogService) SaveLogImage(ctx context.Context, authorID, logID uint64, key string) (*UploadLogImageResult, error) {
	// 验证 key 格式，必须属于该日志图片目录
	if !strings.HasPrefix(key, "logs/") {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "无效的图片 key 格式")
	}

	// 查询日志，验证作者身份
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return nil, err
	}
	if log.AuthorID != authorID {
		return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	// 检查图片数量上限
	var existingImages []string
	if log.Images != "" && log.Images != "[]" {
		if err := json.Unmarshal([]byte(log.Images), &existingImages); err != nil {
			existingImages = []string{}
		}
	}
	if len(existingImages) >= devLogMaxImages {
		return nil, apperrors.CodeError(apperrors.CodeDevLogImageLimit)
	}

	// 将 key 追加到日志图片列表
	existingImages = append(existingImages, key)
	newImagesJSON, _ := json.Marshal(existingImages)
	if err := s.devLogRepo.UpdateLog(ctx, logID, map[string]interface{}{
		"images": string(newImagesJSON),
	}); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	// 返回结果（key 已由前端直传至云存储，此处仅返回元数据）
	return &UploadLogImageResult{
		Key: key,
		URL: "", // URL 由 handler 层中调用 PresignedGetURL 生成
	}, nil
}

// SaveLogVideo 保存日志视频 key（前端已通过预签名 URL 直传后提交）
func (s *devLogService) SaveLogVideo(ctx context.Context, authorID, logID uint64, key string) (*UploadLogVideoResult, error) {
	// 验证 key 格式，必须属于该日志视频目录
	if !strings.HasPrefix(key, "logs/") {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "无效的视频 key 格式")
	}

	// 查询日志，验证作者身份
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return nil, err
	}
	if log.AuthorID != authorID {
		return nil, apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	// 检查视频数量上限
	var existingVideos []string
	if log.Videos != "" && log.Videos != "[]" {
		if err := json.Unmarshal([]byte(log.Videos), &existingVideos); err != nil {
			existingVideos = []string{}
		}
	}
	if len(existingVideos) >= devLogMaxVideos {
		return nil, apperrors.Newf(apperrors.CodeParamInvalid, "每条日志最多上传%d个视频", devLogMaxVideos)
	}

	// 将 key 追加到日志 videos 字段
	existingVideos = append(existingVideos, key)
	newVideosJSON, _ := json.Marshal(existingVideos)
	if err := s.devLogRepo.UpdateLog(ctx, logID, map[string]interface{}{
		"videos": string(newVideosJSON),
	}); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}

	return &UploadLogVideoResult{
		Key:  key,
		URL:  "", // URL 由 handler 层调用 PresignedGetURL 生成
		Size: 0,
	}, nil
}

// ===========================
// Feed 写扩散辅助方法
// ===========================

// asyncFanoutLog 异步将已发布日志写扩散到粉丝的 Feed ZSet。
//
// 仅对粉丝数 < FanoutThreshold（1000）的普通用户执行写扩散；
// 粉丝数 ≥ FanoutThreshold 的大V由读扩散（getFeedLegacy）覆盖。
// 失败不影响主流程，错误仅记录在 feedFanout 内部日志中。
func (s *devLogService) asyncFanoutLog(log *model.DevLog) {
	go func() {
		ctx := context.Background()
		// 查询作者粉丝列表，最多 FanoutThreshold 条
		// 若实际粉丝数 >= FanoutThreshold，说明是大V，写扩散覆盖范围受限，读扩散会补全
		followerIDs, err := s.followRepo.GetFollowerIDsUpToLimit(ctx, log.AuthorID, cache.FanoutThreshold)
		if err != nil {
			// 查询失败则放弃此次写扩散，由读扩散兜底
			return
		}
		// 使用日志创建时间作为 ZSet score（单调递增，保证时间排序正确）
		score := float64(log.CreatedAt.Unix())
		feedFanout(ctx, "log", log.ID, score, followerIDs)
	}()
}

// indexLog 异步将已发布日志写入 ES 索引（失败不影响主流程）
func (s *devLogService) indexLog(ctx context.Context, log *model.DevLog) {
	if s.searcher == nil {
		return
	}
	if err := s.searcher.IndexLog(ctx, search.LogDoc{
		ID:        log.ID,
		Title:     log.Title,
		Content:   log.Content,
		AuthorID:  log.AuthorID,
		ProjectID: log.ProjectID,
		CreatedAt: log.CreatedAt.Unix(),
	}); err != nil {
		logger.Warn("failed to index log", zap.Error(err))
	}
}

// GetMyCollectedLogs 获取当前用户收藏的日志列表（分页）
// 使用批量查询避免 N+1：一次拉取日志记录、一次拉取作者信息、一次拉取点赞/收藏状态。
func (s *devLogService) GetMyCollectedLogs(ctx context.Context, userID uint64, page, pageSize int) ([]*DevLogDetail, int64, error) {
	offset := (page - 1) * pageSize
	logIDs, total, err := s.collectRepo.ListCollectedLogsByUserID(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if len(logIDs) == 0 {
		return []*DevLogDetail{}, total, nil
	}

	// 批量查日志记录（单次 DB）
	logs, err := s.devLogRepo.GetLogsByIDs(ctx, logIDs)
	if err != nil {
		return nil, 0, err
	}
	if len(logs) == 0 {
		return []*DevLogDetail{}, total, nil
	}

	// 批量查作者信息（单次 DB）
	authorIDs := make([]uint64, 0, len(logs))
	for _, l := range logs {
		authorIDs = append(authorIDs, l.AuthorID)
	}
	users, _ := s.userRepo.GetUsersByIDs(ctx, authorIDs)
	userMap := make(map[uint64]*model.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 批量查点赞/收藏状态（各单次 DB）
	allLogIDs := make([]uint64, 0, len(logs))
	for _, l := range logs {
		allLogIDs = append(allLogIDs, l.ID)
	}
	likedMap, _ := s.likeRepo.GetLikedLogIDs(ctx, userID, allLogIDs)
	collectedMap, _ := s.collectRepo.GetCollectedLogIDs(ctx, userID, allLogIDs)

	// 按 logIDs 原始顺序组装结果
	logMap := make(map[uint64]*model.DevLog, len(logs))
	for _, l := range logs {
		logMap[l.ID] = l
	}
	details := make([]*DevLogDetail, 0, len(logIDs))
	for _, id := range logIDs {
		l, ok := logMap[id]
		if !ok {
			continue // 已删除
		}
		d := &DevLogDetail{
			DevLog:      l,
			IsLiked:     likedMap[l.ID],
			IsCollected: collectedMap[l.ID],
		}
		if u, ok := userMap[l.AuthorID]; ok {
			d.AuthorInfo = &DevLogAuthorInfo{
				ID:        u.ID,
				Username:  u.Username,
				Nickname:  u.Nickname,
				AvatarKey: u.AvatarKey,
			}
		}
		details = append(details, d)
	}
	return details, total, nil
}

// GetMyLogs 获取当前用户发布的日志列表（分页）
func (s *devLogService) GetMyLogs(ctx context.Context, userID uint64, page, pageSize int) ([]*model.DevLog, int64, error) {
	offset := (page - 1) * pageSize
	return s.devLogRepo.ListLogsByAuthorID(ctx, userID, offset, pageSize)
}

// DeleteLogVideo 删除日志中的单个视频（通过 对象存储 key，仅作者可操作）

// DeleteLogImage 删除日志中的单个图片（通过 对象存储 key，仅作者可操作）
func (s *devLogService) DeleteLogImage(ctx context.Context, authorID, logID uint64, key string) error {
	if key == "" {
		return apperrors.New(apperrors.CodeParamInvalid, "image key 不能为空")
	}

	// 验证日志存在并校验作者身份
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return err
	}
	if log.AuthorID != authorID {
		return apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	// 解析当前 images 列表
	var images []string
	if log.Images != "" && log.Images != "[]" {
		if err := json.Unmarshal([]byte(log.Images), &images); err != nil {
			images = []string{}
		}
	}

	// 检查 key 是否属于该日志
	found := false
	newImages := make([]string, 0, len(images))
	for _, img := range images {
		if img == key {
			found = true
		} else {
			newImages = append(newImages, img)
		}
	}
	if !found {
		return apperrors.New(apperrors.CodeNotFound, "图片不存在或不属于该日志")
	}

	// 从云存储删除对象（忽略删除失败，以数据库记录为准）
	if err := s.stor.Delete(ctx, key); err != nil {
		logger.Warn("failed to delete storage object", zap.Error(err))
	}

	// 更新 DB
	newImagesJSON, _ := json.Marshal(newImages)
	return s.devLogRepo.UpdateLog(ctx, logID, map[string]interface{}{
		"images": string(newImagesJSON),
	})
}
func (s *devLogService) DeleteLogVideo(ctx context.Context, authorID, logID uint64, key string) error {
	if key == "" {
		return apperrors.New(apperrors.CodeParamInvalid, "video key 不能为空")
	}

	// 验证日志存在并校验作者身份
	log, err := s.devLogRepo.GetLogByID(ctx, logID)
	if err != nil {
		return err
	}
	if log.AuthorID != authorID {
		return apperrors.CodeError(apperrors.CodeDevLogForbidden)
	}

	// 解析当前 videos 列表
	var videos []string
	if log.Videos != "" && log.Videos != "[]" {
		if err := json.Unmarshal([]byte(log.Videos), &videos); err != nil {
			videos = []string{}
		}
	}

	// 检查 key 是否属于该日志
	found := false
	newVideos := make([]string, 0, len(videos))
	for _, v := range videos {
		if v == key {
			found = true
		} else {
			newVideos = append(newVideos, v)
		}
	}
	if !found {
		return apperrors.New(apperrors.CodeNotFound, "视频不存在或不属于该日志")
	}

	// 从云存储删除对象（忽略删除失败，以数据库记录为准）
	if err := s.stor.Delete(ctx, key); err != nil {
		logger.Warn("failed to delete storage object", zap.Error(err))
	}

	// 更新 DB
	newVideosJSON, _ := json.Marshal(newVideos)
	return s.devLogRepo.UpdateLog(ctx, logID, map[string]interface{}{
		"videos": string(newVideosJSON),
	})
}

// populateDevLogDetail fills computed fields (URL conversions, author flattening) on a DevLogDetail.
func (s *devLogService) populateDevLogDetail(ctx context.Context, d *DevLogDetail) {
	if d == nil || d.DevLog == nil {
		return
	}
	log := d.DevLog

	// Convert images JSON → public URLs
	var imageKeys []string
	if log.Images != "" && log.Images != "[]" {
		json.Unmarshal([]byte(log.Images), &imageKeys)
	}
	d.ImageKeys = imageKeys
	d.ImageURLs = make([]string, 0, len(imageKeys))
	for _, key := range imageKeys {
		if key != "" {
			d.ImageURLs = append(d.ImageURLs, s.presignURL(ctx, key))
		}
	}
	if len(d.ImageURLs) > 0 {
		d.CoverURL = d.ImageURLs[0]
	}

	// Convert videos JSON → public URLs
	var videoKeys []string
	if log.Videos != "" && log.Videos != "[]" {
		json.Unmarshal([]byte(log.Videos), &videoKeys)
	}
	d.VideoKeys = videoKeys
	d.VideoURLs = make([]string, 0, len(videoKeys))
	for _, key := range videoKeys {
		if key != "" {
			d.VideoURLs = append(d.VideoURLs, s.presignURL(ctx, key))
		}
	}

	// Flatten author info for frontend compatibility
	if d.AuthorInfo != nil {
		d.AuthorNickname = d.AuthorInfo.Nickname
		d.AuthorUsername = d.AuthorInfo.Username
		d.AuthorAvatarURL = d.AuthorInfo.AvatarURL
	}

	// Project info (best-effort)
	if log.ProjectID > 0 {
		if proj, err := s.projectRepo.GetProjectByID(ctx, log.ProjectID); err == nil && proj != nil {
			d.ProjectName = proj.Name
			d.ProjectSlug = proj.Slug
			if proj.CoverKey != "" {
				d.ProjectCoverURL = s.presignURL(ctx, proj.CoverKey)
			}
		}
	}
}
