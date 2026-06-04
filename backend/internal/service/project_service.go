// Package service 提供项目系统的业务逻辑层。
// 本文件包含项目相关的所有业务逻辑：
//   - 项目创建/编辑/删除
//   - 图片 key 保存（封面、截图，前端直传后提交 key）
//   - 成员管理
//   - 项目状态管理与时间轴
//   - 里程碑管理
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/mq"
	"github.com/gamero/gamero/pkg/search"
	"github.com/gamero/gamero/pkg/storage"
	"github.com/google/uuid"
	"github.com/mozillazg/go-pinyin"
	"go.uber.org/zap"
)

// 项目限制常量
const (
	// maxScreenshots 截图最多10张
	maxScreenshots = 10
	// maxStyleTags 风格标签最多5个
	maxStyleTags = 5
	// projectPresignExpiry 预签名 URL 有效期 24 小时
	projectPresignExpiry = 24 * time.Hour
	// slugRandLen slug 冲突时追加的随机字符长度
	slugRandLen = 4
)

// 允许的图片 MIME 类型及对应扩展名
var allowedImageMIME = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// 非字母数字连字符的字符替换正则
var slugInvalidCharsRegexp = regexp.MustCompile(`[^a-z0-9\-]+`)

// 连续连字符替换正则
var slugMultiDashRegexp = regexp.MustCompile(`-{2,}`)

// ===========================
// 请求/响应 DTO
// ===========================

// CreateProjectReq 创建项目请求
type CreateProjectReq struct {
	Name        string   `json:"name" binding:"required,min=1,max=128"`
	Description string   `json:"description"`
	Genre       string   `json:"genre"`
	StyleTags   []string `json:"style_tags"`
	Status      string   `json:"status"`
	Visibility  string   `json:"visibility"` // public（默认）/ private
	DemoURL     string   `json:"demo_url"`
	StoreURL    string   `json:"store_url"`
}

// UpdateProjectReq 更新项目请求（使用指针区分零值和未设置）
type UpdateProjectReq struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Genre       *string  `json:"genre"`
	StyleTags   []string `json:"style_tags"`
	Visibility  *string  `json:"visibility"` // public / private
	DemoURL     *string  `json:"demo_url"`
	StoreURL    *string  `json:"store_url"`
	VideoKeys   []string `json:"video_keys"` // 视频 key 列表（最多3个）
}

// UpdateStatusReq 更新项目状态请求
type UpdateStatusReq struct {
	Status string `json:"status" binding:"required"`
}

// AddMemberReq 添加成员请求
type AddMemberReq struct {
	UserID       uint64 `json:"user_id" binding:"required"`
	Role         string `json:"role" binding:"required"`
	Contribution string `json:"contribution"`
}

// UpdateMemberReq 更新成员请求
// UserID 是目标成员的用户 ID
type UpdateMemberReq struct {
	Role         string `json:"role" binding:"required"`
	Contribution string `json:"contribution"`
}

// ProjectTaskReq 创建/更新项目任务请求
type ProjectTaskReq struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	AssigneeID  *uint64 `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
}

type ProjectTaskDetail struct {
	ID               uint64                    `json:"id"`
	ProjectID        uint64                    `json:"project_id"`
	CreatorID        uint64                    `json:"creator_id"`
	AssigneeID       *uint64                   `json:"assignee_id,omitempty"`
	Title            string                    `json:"title"`
	Description      string                    `json:"description,omitempty"`
	Status           model.ProjectTaskStatus   `json:"status"`
	Priority         model.ProjectTaskPriority `json:"priority"`
	DueDate          *time.Time                `json:"due_date,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
	CreatorNickname  string                    `json:"creator_nickname,omitempty"`
	AssigneeNickname string                    `json:"assignee_nickname,omitempty"`
}

// MemberDetail 成员详情（含用户基本信息）
type MemberDetail struct {
	ID           uint64                  `json:"id"`
	ProjectID    uint64                  `json:"project_id"`
	UserID       uint64                  `json:"user_id"`
	Role         model.ProjectMemberRole `json:"role"`
	Contribution string                  `json:"contribution,omitempty"`
	JoinedAt     time.Time               `json:"joined_at"`
	LeftAt       *time.Time              `json:"left_at,omitempty"`
	Username     string                  `json:"username,omitempty"`
	Nickname     string                  `json:"nickname"`
	AvatarKey    string                  `json:"-"`
	AvatarURL    string                  `json:"avatar_url,omitempty"`
}

// ProjectDetail 项目详情响应（显式字段，不直接内嵌 model.Project 以避免泄露内部字段）
type ProjectDetail struct {
	ID              uint64                  `json:"id"`
	OwnerID         uint64                  `json:"owner_id"`
	Name            string                  `json:"name"`
	Slug            string                  `json:"slug"`
	Description     string                  `json:"description"`
	Genre           model.ProjectGenre      `json:"genre"`
	StyleTags       []string                `json:"style_tags"`
	StyleTagsParsed []string                `json:"style_tags_parsed"`
	Engine          string                  `json:"engine,omitempty"`
	Platform        []string                `json:"platform,omitempty"`
	Status          model.ProjectStatus     `json:"status"`
	Visibility      model.ProjectVisibility `json:"visibility"`
	CoverURL        string                  `json:"cover_url,omitempty"`
	ScreenshotURLs  []string                `json:"screenshot_urls"`
	ScreenshotKeys  []string                `json:"screenshot_keys"`
	DemoURL         string                  `json:"demo_url,omitempty"`
	StoreURL        string                  `json:"store_url,omitempty"`
	VideoURLs       []string                `json:"video_urls"`
	FollowerCount   int                     `json:"follower_count"`
	ReleaseCount    int                     `json:"release_count"`
	TotalDownloads  int                     `json:"total_downloads"`
	IsFollowed      bool                    `json:"is_followed"`
	IsCollected     bool                    `json:"is_collected"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	Members         []*MemberDetail         `json:"members"`
}

// ProjectListItem 项目列表项（简化版）
type ProjectListItem struct {
	ID            uint64              `json:"id"`
	OwnerID       uint64              `json:"owner_id"`
	Name          string              `json:"name"`
	Slug          string              `json:"slug"`
	Description   string              `json:"description"`
	Genre         model.ProjectGenre  `json:"genre"`
	StyleTags     []string            `json:"style_tags"`
	Status        model.ProjectStatus `json:"status"`
	CoverURL      string              `json:"cover_url,omitempty"`
	FollowerCount int                 `json:"follower_count"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// ListProjectsParams 项目列表查询参数
type ListProjectsParams struct {
	Page     int    // 页码（从1开始）
	PageSize int    // 每页大小
	Status   string // 按状态筛选（可选）
	Genre    string // 按类型筛选（可选）
	Keyword  string // 关键词搜索（可选）
	OwnerID  uint64 // 按创建者筛选（可选）
	SortBy   string // 排序字段：newest（默认）/ hottest / updated
}

// ===========================
// Service 接口
// ===========================

// ProjectService 项目业务逻辑接口
type ProjectService interface {
	// SetBroker 注入消息队列（用于 Feed 扩散事件），可在 NewProjectService 后调用
	SetBroker(b mq.Broker)
	// CreateProject 创建项目，自动生成 slug，写入创建时间轴，添加 owner 成员
	CreateProject(ctx context.Context, ownerID uint64, req *CreateProjectReq) (*model.Project, error)
	// GetProject 获取项目详情（by ID），含成员/里程碑/预签名URL
	GetProject(ctx context.Context, projectID, viewerID uint64) (*ProjectDetail, error)
	// GetProjectBySlug 获取项目详情（by slug）
	GetProjectBySlug(ctx context.Context, slug string, viewerID uint64) (*ProjectDetail, error)
	// UpdateProject 更新项目信息（仅 owner 可操作）
	UpdateProject(ctx context.Context, ownerID, projectID uint64, req *UpdateProjectReq) (*model.Project, error)
	// UpdateStatus 更新项目状态，写入时间轴（仅 owner 可操作）
	UpdateStatus(ctx context.Context, ownerID, projectID uint64, newStatus string) (*model.Project, error)
	// DeleteProject 软删除项目（仅 owner 可操作）
	DeleteProject(ctx context.Context, ownerID, projectID uint64) error
	// SaveCover 保存封面 key（前端直传后提交，仅 owner 可操作）
	SaveCover(ctx context.Context, ownerID, projectID uint64, key string) error
	// SaveScreenshot 保存截图 key（前端直传后提交，仅 owner 可操作）
	SaveScreenshot(ctx context.Context, ownerID, projectID uint64, key string) error
	// DeleteScreenshot 删除截图（仅 owner 可操作）
	DeleteScreenshot(ctx context.Context, ownerID, projectID uint64, key string) error
	// SaveVideo 保存视频 key（前端直传后提交，仅 owner 可操作，最多3个）
	SaveVideo(ctx context.Context, ownerID, projectID uint64, key string) error
	// DeleteVideo 删除视频（仅 owner 可操作）
	DeleteVideo(ctx context.Context, ownerID, projectID uint64, key string) error
	// AddMember 添加成员（仅 owner 可操作）
	AddMember(ctx context.Context, ownerID, projectID uint64, req *AddMemberReq) (*model.ProjectMember, error)
	// RemoveMember 移除成员（仅 owner 可操作），targetUserID 为被移除成员的 user_id
	RemoveMember(ctx context.Context, ownerID, projectID, targetUserID uint64) error
	// LeaveProject 成员主动退出项目（非 owner 可操作，记录离职时间）
	LeaveProject(ctx context.Context, userID, projectID uint64) error
	// UpdateMember 更新成员角色/贡献（仅 owner 可操作），targetUserID 为目标成员的 user_id
	UpdateMember(ctx context.Context, ownerID, projectID, targetUserID uint64, req *UpdateMemberReq) error
	// ListProjects 查询项目列表（带分页筛选，封面返回预签名URL）
	ListProjects(ctx context.Context, params ListProjectsParams) ([]*ProjectListItem, int64, error)
	GetTimeline(ctx context.Context, projectID uint64, page, pageSize int) ([]*model.ProjectTimeline, int64, error)
	GetProjectMembers(ctx context.Context, projectID uint64) ([]*model.ProjectMember, error)
	// GetUserProjects 获取指定用户名的公开项目列表（分页）
	GetUserProjects(ctx context.Context, userID uint64, page, pageSize int) ([]*ProjectListItem, int64, error)
	// CollectProject 收藏项目
	CollectProject(ctx context.Context, userID, projectID uint64) error
	// UncollectProject 取消收藏
	UncollectProject(ctx context.Context, userID, projectID uint64) error
	// GetMyCollectedProjects 获取我收藏的项目列表（分页）
	GetMyCollectedProjects(ctx context.Context, userID uint64, page, pageSize int) ([]*ProjectListItem, int64, error)
	ListTasks(ctx context.Context, userID, projectID uint64) ([]*ProjectTaskDetail, error)
	CreateTask(ctx context.Context, userID, projectID uint64, req *ProjectTaskReq) (*ProjectTaskDetail, error)
	UpdateTask(ctx context.Context, userID, projectID, taskID uint64, req *ProjectTaskReq) (*ProjectTaskDetail, error)
	DeleteTask(ctx context.Context, userID, projectID, taskID uint64) error

	// SetDevLogRepository 注入开发日志仓库（可在 NewProjectService 后调用，用于统计版本发布数）
	SetDevLogRepository(repo repository.DevLogRepository)
	// TransferOwner 转让项目所有权（仅当前 owner 可操作）
	TransferOwner(ctx context.Context, ownerID, projectID, newOwnerID uint64) error
}

// ===========================
// Service 实现
// ===========================

// projectService 项目业务逻辑实现
type projectService struct {
	repo       repository.ProjectRepository
	userRepo   repository.UserRepository
	followRepo repository.FollowRepository // 用于查询项目关注者 ID
	teamRepo   repository.TeamRepository   // 用于写入通知记录
	devLogRepo repository.DevLogRepository // 用于统计版本发布数（可为 nil）
	*NotificationClient
	storage  *storage.Client
	searcher search.Searcher
	broker   mq.Broker // 消息队列（可为 nil）
}

// NewProjectService 创建项目服务实例
func NewProjectService(
	repo repository.ProjectRepository,
	userRepo repository.UserRepository,
	followRepo repository.FollowRepository,
	teamRepo repository.TeamRepository,
	stor *storage.Client,
	searcher search.Searcher,
) ProjectService {
	svc := &projectService{
		repo:               repo,
		userRepo:           userRepo,
		followRepo:         followRepo,
		teamRepo:           teamRepo,
		storage:            stor,
		searcher:           searcher,
		NotificationClient: &NotificationClient{},
	}
	return svc
}

// SetBroker 注入消息队列 Broker（在 router 初始化后调用）
func (s *projectService) SetBroker(b mq.Broker) {
	s.broker = b
}

// SetDevLogRepository 注入开发日志仓库（在 NewProjectService 后调用）
func (s *projectService) SetDevLogRepository(repo repository.DevLogRepository) {
	s.devLogRepo = repo
}

// ===========================
// 工具函数：slug 生成
// ===========================

// generateSlug 将项目名生成 URL 友好的 slug
// 中文字符使用拼音转换，英文字符保留并转小写
func generateSlug(name string) string {
	// 初始化拼音转换参数（不带声调）
	a := pinyin.NewArgs()
	a.Style = pinyin.Normal

	var parts []string
	var buf strings.Builder

	for _, r := range name {
		if unicode.Is(unicode.Han, r) {
			// 中文字符：先刷新缓冲区
			if buf.Len() > 0 {
				parts = append(parts, buf.String())
				buf.Reset()
			}
			// 使用 go-pinyin 转换单个汉字
			pys := pinyin.SinglePinyin(r, a)
			if len(pys) > 0 {
				parts = append(parts, pys[0])
			}
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// ASCII 字母/数字：转小写后写入缓冲区
			buf.WriteRune(unicode.ToLower(r))
		} else {
			// 其他字符（空格、标点等）作为分隔符
			if buf.Len() > 0 {
				parts = append(parts, buf.String())
				buf.Reset()
			}
		}
	}

	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}

	slug := strings.Join(parts, "-")
	slug = slugInvalidCharsRegexp.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	slug = slugMultiDashRegexp.ReplaceAllString(slug, "-")

	// 兜底：slug 为空时使用随机字符串
	if slug == "" {
		slug = randomString(8)
	}

	return slug
}

// randomString 生成指定长度的随机小写字母+数字字符串
func randomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	id := uuid.New()
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[int(id[i%16])%len(chars)]
	}
	return string(b)
}

// generateUniqueSlug 生成唯一 slug，冲突时追加随机后缀
func (s *projectService) generateUniqueSlug(ctx context.Context, name string) (string, error) {
	baseSlug := generateSlug(name)
	slug := baseSlug

	for i := 0; i < 10; i++ {
		exists, err := s.repo.ExistsSlug(ctx, slug)
		if err != nil {
			return "", apperrors.Wrap(apperrors.CodeInternalError, err)
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%s", baseSlug, randomString(slugRandLen))
	}

	return "", apperrors.New(apperrors.CodeInternalError, "生成唯一 slug 失败")
}

// ===========================
// 工具函数：JSON 序列化
// ===========================

// parseStringSlice 解析 JSON 字符串为 []string
func parseStringSlice(jsonStr string) []string {
	var result []string
	if jsonStr == "" || jsonStr == "null" {
		return []string{}
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return []string{}
	}
	return result
}

// encodeStringSlice 将 []string 序列化为 JSON 字符串
func encodeStringSlice(items []string) string {
	if items == nil {
		items = []string{}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

// ===========================
// 工具函数：权限验证
// ===========================

// requireOwner 验证操作者是否为项目负责人，否则返回 403
func (s *projectService) requireOwner(ctx context.Context, projectID, userID uint64) error {
	isOwner, err := s.repo.IsOwner(ctx, projectID, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if !isOwner {
		return apperrors.CodeError(apperrors.CodeProjectForbidden)
	}
	return nil
}

// isMember 检查 userID 是否为 projectID 的成员（含 owner）
func (s *projectService) isMember(ctx context.Context, projectID, userID uint64) bool {
	members, err := s.repo.GetMembers(ctx, projectID)
	if err != nil {
		return false
	}
	for _, m := range members {
		if m.UserID == userID {
			return true
		}
	}
	return false
}

// ===========================
// 工具函数：项目存在性检查
// ===========================

// mustGetProject 获取项目，若不存在则返回 404
func (s *projectService) mustGetProject(ctx context.Context, projectID uint64) (*model.Project, error) {
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err // repository 层已返回 AppError
	}
	return project, nil
}

// ===========================
// 工具函数：预签名 URL
// ===========================

// getPublicURL 获取对象存储 URL（24h 预签名 URL），key 为空则返回空字符串
func (s *projectService) getPublicURL(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	presigned, err := s.storage.PresignedGetURL(ctx, key, 24*time.Hour)
	if err != nil {
		logger.Warn("failed to generate presigned URL for project media", zap.String("key", key), zap.Error(err))
		return s.storage.GetPublicURL(key)
	}
	return presigned
}

// ===========================
// 工具函数：写入时间轴
// ===========================

// writeTimelineEvent 写入时间轴事件（不影响主流程，错误静默处理）
func (s *projectService) writeTimelineEvent(ctx context.Context, projectID uint64, eventType model.TimelineEventType, title, content string, metadata map[string]interface{}) {
	metadataStr := "{}"
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			metadataStr = string(b)
		}
	}
	event := &model.ProjectTimeline{
		ProjectID: projectID,
		Type:      eventType,
		Title:     title,
		Content:   content,
		Metadata:  json.RawMessage(metadataStr),
	}
	if err := s.repo.CreateTimelineEvent(ctx, event); err != nil {
		logger.Warn("failed to create timeline event", zap.Error(err))
	}
}

// ===========================
// 工具函数：通知预留
// ===========================

// notifyFollowers 向项目的关注者批量写入状态变更通知。
// 最多通知前 500 位关注者（防止关注量极大时单次操作耗时过长）。
// 通知写入失败不影响主流程。
func (s *projectService) notifyFollowers(ctx context.Context, projectID uint64, message string) {
	const maxNotify = 500
	followerIDs, err := s.followRepo.GetProjectFollowerIDsUpToLimit(ctx, projectID, maxNotify)
	if err != nil || len(followerIDs) == 0 {
		return
	}

	if s.Svc == nil {
		return
	}

	meta := map[string]interface{}{"project_id": projectID}
	reqs := make([]*SendNotificationReq, 0, len(followerIDs))
	for _, uid := range followerIDs {
		reqs = append(reqs, &SendNotificationReq{
			UserID:   uid,
			Type:     model.NotificationTypeProjectStatusChanged,
			Title:    "关注的项目有新动态",
			Content:  message,
			Metadata: meta,
		})
	}
	s.Svc.SendBatch(ctx, reqs)
}

// ===========================
// 工具函数：构建成员详情
// ===========================

// buildMemberDetails 构建含用户基本信息的成员详情列表（批量查询，避免 N+1）
func (s *projectService) buildMemberDetails(ctx context.Context, members []*model.ProjectMember) []*MemberDetail {
	// 批量收集 UserID
	userIDs := make([]uint64, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}
	// 批量查询用户基本信息
	userMap := make(map[uint64]*model.User)
	if users, err := s.userRepo.GetUsersByIDs(ctx, userIDs); err == nil {
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	details := make([]*MemberDetail, 0, len(members))
	for _, m := range members {
		detail := &MemberDetail{
			ID:           m.ID,
			ProjectID:    m.ProjectID,
			UserID:       m.UserID,
			Role:         m.Role,
			Contribution: m.Contribution,
			JoinedAt:     m.JoinedAt,
			LeftAt:       m.LeftAt,
		}
		if user, ok := userMap[m.UserID]; ok {
			detail.Nickname = user.Nickname
			detail.AvatarKey = user.AvatarKey
			if user.AvatarKey != "" {
				detail.AvatarURL = s.getPublicURL(ctx, user.AvatarKey)
			}
		}
		details = append(details, detail)
	}
	return details
}

// ===========================
// 工具函数：构建项目详情
// ===========================

// buildProjectDetail 将 Project 模型组装为 ProjectDetail 响应
func (s *projectService) buildProjectDetail(ctx context.Context, project *model.Project, viewerID uint64) (*ProjectDetail, error) {
	styleTags := parseStringSlice(project.StyleTags)
	screenshotKeys := parseStringSlice(project.ScreenshotKeys)

	// 生成封面和截图预签名 URL
	coverURL := s.getPublicURL(ctx, project.CoverKey)
	screenshotURLs := make([]string, 0, len(screenshotKeys))
	for _, key := range screenshotKeys {
		screenshotURLs = append(screenshotURLs, s.getPublicURL(ctx, key))
	}

	// 生成视频公开 URL
	videoKeys := parseStringSlice(project.VideoKeys)
	videoURLs := make([]string, 0, len(videoKeys))
	for _, key := range videoKeys {
		videoURLs = append(videoURLs, s.getPublicURL(ctx, key))
	}

	// 查询成员列表
	members, err := s.repo.GetMembers(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	memberDetails := s.buildMemberDetails(ctx, members)

	detail := &ProjectDetail{
		ID:              project.ID,
		OwnerID:         project.OwnerID,
		Name:            project.Name,
		Slug:            project.Slug,
		Description:     project.Description,
		Genre:           project.Genre,
		StyleTags:       styleTags,
		StyleTagsParsed: styleTags,
		Status:          project.Status,
		Visibility:      project.Visibility,
		CoverURL:        coverURL,
		ScreenshotURLs:  screenshotURLs,
		ScreenshotKeys:  screenshotKeys,
		DemoURL:         project.DemoURL,
		StoreURL:        project.StoreURL,
		VideoURLs:       videoURLs,
		FollowerCount:   project.FollowerCount,
		ReleaseCount:    0,
		TotalDownloads:  0,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
		IsFollowed:      false,
		IsCollected:     false,
		Members:         memberDetails,
	}

	// 查询当前用户的关注/收藏状态
	if viewerID > 0 {
		if s.followRepo != nil {
			if followed, err := s.followRepo.IsFollowingProject(ctx, viewerID, project.ID); err == nil {
				detail.IsFollowed = followed
			}
		}
	}

	// 查询版本发布数量和下载总数（如果 devLogRepo 已注入）
	if s.devLogRepo != nil {
		if rc, err := s.devLogRepo.CountReleasesByProjectID(ctx, project.ID); err == nil {
			detail.ReleaseCount = int(rc)
		}
		if dl, err := s.devLogRepo.SumDownloadsByProjectID(ctx, project.ID); err == nil {
			detail.TotalDownloads = int(dl)
		}
	}

	return detail, nil
}

// ===========================
// Service 方法实现
// ===========================

// CreateProject 创建项目
func (s *projectService) CreateProject(ctx context.Context, ownerID uint64, req *CreateProjectReq) (*model.Project, error) {
	// 验证游戏类型
	if req.Genre != "" && !model.ValidProjectGenres[model.ProjectGenre(req.Genre)] {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "无效的游戏类型")
	}

	// 验证风格标签数量和内容
	if len(req.StyleTags) > maxStyleTags {
		return nil, apperrors.Newf(apperrors.CodeParamInvalid, "风格标签最多 %d 个", maxStyleTags)
	}
	for _, tag := range req.StyleTags {
		if !model.ValidProjectStyleTags[model.ProjectStyleTag(tag)] {
			return nil, apperrors.Newf(apperrors.CodeParamInvalid, "无效的风格标签: %s", tag)
		}
	}

	// 验证状态
	status := model.ProjectStatusPreparing
	if req.Status != "" {
		if !model.ValidProjectStatuses[model.ProjectStatus(req.Status)] {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "无效的项目状态")
		}
		status = model.ProjectStatus(req.Status)
	}

	// 生成唯一 slug
	slug, err := s.generateUniqueSlug(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	// 敏感词检测
	if filter.Contains(req.Name) || filter.Contains(req.Description) {
		return nil, apperrors.New(apperrors.CodeContentViolation, "项目名称或简介包含敏感词，请修改后重试")
	}

	// 验证可见性
	visibility := model.ProjectVisibilityPublic
	if req.Visibility == string(model.ProjectVisibilityPrivate) {
		visibility = model.ProjectVisibilityPrivate
	}

	project := &model.Project{
		OwnerID:        ownerID,
		Name:           req.Name,
		Slug:           slug,
		Description:    req.Description,
		Genre:          model.ProjectGenre(req.Genre),
		StyleTags:      encodeStringSlice(req.StyleTags),
		Status:         status,
		Visibility:     visibility,
		ScreenshotKeys: "[]",
		VideoKeys:      "[]",
		DemoURL:        req.DemoURL,
		StoreURL:       req.StoreURL,
		FollowerCount:  0,
	}

	if err := s.repo.CreateProject(ctx, project); err != nil {
		return nil, err
	}

	// 添加 owner 为项目成员
	ownerMember := &model.ProjectMember{
		ProjectID:    project.ID,
		UserID:       ownerID,
		Role:         model.ProjectMemberRoleOwner,
		Contribution: "项目负责人",
		JoinedAt:     time.Now(),
	}
	if err := s.repo.AddMember(ctx, ownerMember); err != nil {
		return nil, err
	}

	// 写入"项目创建"时间轴事件
	s.writeTimelineEvent(
		ctx,
		project.ID,
		model.TimelineEventCreated,
		fmt.Sprintf("项目「%s」正式创立", project.Name),
		"项目已创建，开始游戏开发之旅！",
		map[string]interface{}{"initial_status": string(project.Status)},
	)

	// 异步写入 ES 索引
	go s.indexProject(context.Background(), project)

	return project, nil
}

// GetProject 获取项目详情（by ID）
func (s *projectService) GetProject(ctx context.Context, projectID, viewerID uint64) (*ProjectDetail, error) {
	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.buildProjectDetail(ctx, project, viewerID)
}

// GetProjectBySlug 获取项目详情（by slug）
func (s *projectService) GetProjectBySlug(ctx context.Context, slug string, viewerID uint64) (*ProjectDetail, error) {
	project, err := s.repo.GetProjectBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	// 私密项目：仅 owner 和成员可查看
	if project.Visibility == model.ProjectVisibilityPrivate {
		if viewerID == 0 || (viewerID != project.OwnerID && !s.isMember(ctx, project.ID, viewerID)) {
			return nil, apperrors.CodeError(apperrors.CodeNotFound)
		}
	}
	return s.buildProjectDetail(ctx, project, viewerID)
}

// UpdateProject 更新项目信息（仅 owner）
func (s *projectService) UpdateProject(ctx context.Context, ownerID, projectID uint64, req *UpdateProjectReq) (*model.Project, error) {
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}

	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 构建更新 map（只更新请求中携带的字段）
	updates := make(map[string]interface{})

	if req.Name != nil {
		if len(*req.Name) == 0 || len(*req.Name) > 128 {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "项目名称长度须在 1-128 之间")
		}
		if filter.Contains(*req.Name) {
			return nil, apperrors.New(apperrors.CodeContentViolation, "项目名称包含敏感词，请修改后重试")
		}
		updates["name"] = *req.Name
	}

	if req.Description != nil {
		if filter.Contains(*req.Description) {
			return nil, apperrors.New(apperrors.CodeContentViolation, "项目简介包含敏感词，请修改后重试")
		}
		updates["description"] = *req.Description
	}

	if req.Genre != nil {
		if *req.Genre != "" && !model.ValidProjectGenres[model.ProjectGenre(*req.Genre)] {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "无效的游戏类型")
		}
		updates["genre"] = *req.Genre
	}

	if req.StyleTags != nil {
		if len(req.StyleTags) > maxStyleTags {
			return nil, apperrors.Newf(apperrors.CodeParamInvalid, "风格标签最多 %d 个", maxStyleTags)
		}
		for _, tag := range req.StyleTags {
			if !model.ValidProjectStyleTags[model.ProjectStyleTag(tag)] {
				return nil, apperrors.Newf(apperrors.CodeParamInvalid, "无效的风格标签: %s", tag)
			}
		}
		updates["style_tags"] = encodeStringSlice(req.StyleTags)
	}

	if req.DemoURL != nil {
		updates["demo_url"] = *req.DemoURL
	}

	if req.StoreURL != nil {
		updates["store_url"] = *req.StoreURL
	}

	if req.VideoKeys != nil {
		if len(req.VideoKeys) > 3 {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "视频最多3个")
		}
		updates["video_keys"] = encodeStringSlice(req.VideoKeys)
	}

	if req.Visibility != nil {
		v := model.ProjectVisibility(*req.Visibility)
		if v != model.ProjectVisibilityPublic && v != model.ProjectVisibilityPrivate {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "visibility 仅支持 public / private")
		}
		updates["visibility"] = v
	}

	if len(updates) == 0 {
		return project, nil
	}

	if err := s.repo.UpdateProject(ctx, projectID, updates); err != nil {
		return nil, err
	}

	// 重新查询返回最新数据
	updated, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	// 异步更新 ES 索引
	go s.indexProject(context.Background(), updated)
	return updated, nil
}

// UpdateStatus 更新项目状态（仅 owner），写入时间轴，通知关注者
func (s *projectService) UpdateStatus(ctx context.Context, ownerID, projectID uint64, newStatus string) (*model.Project, error) {
	if !model.ValidProjectStatuses[model.ProjectStatus(newStatus)] {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "无效的项目状态")
	}

	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}

	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	oldStatus := project.Status
	if string(oldStatus) == newStatus {
		return project, nil
	}

	// 更新状态
	if err := s.repo.UpdateProject(ctx, projectID, map[string]interface{}{
		"status": newStatus,
	}); err != nil {
		return nil, err
	}

	// 状态中文标签映射
	statusLabels := map[model.ProjectStatus]string{
		model.ProjectStatusPreparing:  "立项中",
		model.ProjectStatusDeveloping: "开发中",
		model.ProjectStatusPlayable:   "可玩Demo",
		model.ProjectStatusLaunched:   "已上线",
		model.ProjectStatusPaused:     "暂停",
		model.ProjectStatusAbandoned:  "放弃",
	}
	oldLabel := statusLabels[oldStatus]
	newLabel := statusLabels[model.ProjectStatus(newStatus)]

	// 确定事件类型
	eventType := model.TimelineEventStatusChanged
	if model.ProjectStatus(newStatus) == model.ProjectStatusLaunched {
		eventType = model.TimelineEventLaunched
	} else if model.ProjectStatus(newStatus) == model.ProjectStatusAbandoned {
		eventType = model.TimelineEventEnded
	}

	// 写入状态变更时间轴
	s.writeTimelineEvent(
		ctx,
		project.ID,
		eventType,
		fmt.Sprintf("项目状态更新：%s → %s", oldLabel, newLabel),
		fmt.Sprintf("项目状态由「%s」变更为「%s」", oldLabel, newLabel),
		map[string]interface{}{
			"old_status": string(oldStatus),
			"new_status": newStatus,
		},
	)

	// Demo 发布专项事件
	if model.ProjectStatus(newStatus) == model.ProjectStatusPlayable {
		s.writeTimelineEvent(
			ctx,
			project.ID,
			model.TimelineEventDemoPublished,
			"可玩 Demo 发布",
			"项目已发布可玩 Demo，欢迎体验并反馈！",
			nil,
		)
	}

	// 通知关注者（异步，不阻塞主流程）
	notifyMsg := fmt.Sprintf("项目「%s」状态更新为「%s」", project.Name, newLabel)
	go s.notifyFollowers(context.Background(), project.ID, notifyMsg)

	return s.mustGetProject(ctx, projectID)
}

// DeleteProject 软删除项目（仅 owner）
func (s *projectService) DeleteProject(ctx context.Context, ownerID, projectID uint64) error {
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}
	if err := s.repo.SoftDeleteProject(ctx, projectID); err != nil {
		return err
	}
	// 异步从 ES 删除索引
	go func() {
		if delErr := s.searcher.DeleteProject(context.Background(), projectID); delErr != nil {
			logger.Get().Warn("search: failed to delete project from ES",
				zap.Uint64("project_id", projectID), zap.Error(delErr))
		}
	}()
	return nil
}

// SaveCover 保存封面 key（前端已通过预签名 URL 直传后提交）
func (s *projectService) SaveCover(ctx context.Context, ownerID, projectID uint64, key string) error {
	// 验证 owner 权限
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}
	// 验证 key 格式，必须以 projects/ 开头
	if !strings.HasPrefix(key, "projects/") {
		return apperrors.New(apperrors.CodeParamInvalid, "无效的封面 key 格式")
	}

	// 查询旧封面 key以删除
	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return err
	}
	oldKey := project.CoverKey

	// 更新封面 key
	if err := s.repo.UpdateProject(ctx, projectID, map[string]interface{}{"cover_key": key}); err != nil {
		return err
	}

	// 异步删除旧封面（静默忽略错误）
	if oldKey != "" && oldKey != key {
		go func() {
			if err := s.storage.Delete(ctx, oldKey); err != nil {
				logger.Warn("failed to delete old cover", zap.String("key", oldKey), zap.Error(err))
			}
		}()
	}
	return nil
}

// SaveScreenshot 保存截图 key（前端已通过预签名 URL 直传后提交）
func (s *projectService) SaveScreenshot(ctx context.Context, ownerID, projectID uint64, key string) error {
	// 验证 owner 权限
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}
	// 验证 key 格式必须属于该项目截图目录
	if !strings.HasPrefix(key, "projects/") {
		return apperrors.New(apperrors.CodeParamInvalid, "无效的截图 key 格式")
	}

	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return err
	}

	// 检查截图数量上限
	existingKeys := parseStringSlice(project.ScreenshotKeys)
	if len(existingKeys) >= maxScreenshots {
		return apperrors.Newf(apperrors.CodeParamInvalid, "截图最多 %d 张", maxScreenshots)
	}

	// 将新 key 追加到数据库
	existingKeys = append(existingKeys, key)
	if err := s.repo.UpdateProject(ctx, projectID, map[string]interface{}{
		"screenshot_keys": encodeStringSlice(existingKeys),
	}); err != nil {
		return err
	}
	return nil
}

// DeleteScreenshot 删除截图
func (s *projectService) DeleteScreenshot(ctx context.Context, ownerID, projectID uint64, key string) error {
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}

	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return err
	}

	existingKeys := parseStringSlice(project.ScreenshotKeys)
	found := false
	newKeys := make([]string, 0, len(existingKeys))
	for _, k := range existingKeys {
		if k == key {
			found = true
		} else {
			newKeys = append(newKeys, k)
		}
	}
	if !found {
		return apperrors.New(apperrors.CodeResourceNotFound, "截图不存在")
	}

	// 从云存储删除文件（静默处理错误）
	if err := s.storage.Delete(ctx, key); err != nil {
		logger.Warn("failed to delete storage object", zap.Error(err))
	}

	return s.repo.UpdateProject(ctx, projectID, map[string]interface{}{
		"screenshot_keys": encodeStringSlice(newKeys),
	})
}

// SaveVideo 保存视频 key（前端已通过预签名 URL 直传后提交）
func (s *projectService) SaveVideo(ctx context.Context, ownerID, projectID uint64, key string) error {
	// 验证 owner 权限
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}
	// 验证 key 格式必须属于该项目视频目录
	if !strings.HasPrefix(key, "projects/") {
		return apperrors.New(apperrors.CodeParamInvalid, "无效的视频 key 格式")
	}

	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return err
	}

	// 检查视频数量上限
	existingKeys := parseStringSlice(project.VideoKeys)
	if len(existingKeys) >= 3 {
		return apperrors.Newf(apperrors.CodeParamInvalid, "视频最多 3 个")
	}

	// 将新 key 追加到数据库
	existingKeys = append(existingKeys, key)
	if err := s.repo.UpdateProject(ctx, projectID, map[string]interface{}{
		"video_keys": encodeStringSlice(existingKeys),
	}); err != nil {
		return err
	}
	return nil
}

// DeleteVideo 删除视频（仅 owner 可操作）
func (s *projectService) DeleteVideo(ctx context.Context, ownerID, projectID uint64, key string) error {
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}

	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return err
	}

	existingKeys := parseStringSlice(project.VideoKeys)
	found := false
	newKeys := make([]string, 0, len(existingKeys))
	for _, k := range existingKeys {
		if k == key {
			found = true
		} else {
			newKeys = append(newKeys, k)
		}
	}
	if !found {
		return apperrors.New(apperrors.CodeResourceNotFound, "视频不存在")
	}

	// 从云存储删除文件（静默处理错误）
	if err := s.storage.Delete(ctx, key); err != nil {
		logger.Warn("failed to delete storage object", zap.Error(err))
	}

	return s.repo.UpdateProject(ctx, projectID, map[string]interface{}{
		"video_keys": encodeStringSlice(newKeys),
	})
}

// AddMember 添加项目成员（仅 owner）
func (s *projectService) AddMember(ctx context.Context, ownerID, projectID uint64, req *AddMemberReq) (*model.ProjectMember, error) {
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}

	// 验证角色合法性（不允许直接设置 owner 角色）
	if !model.ValidProjectMemberRoles[model.ProjectMemberRole(req.Role)] {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "无效的成员角色")
	}

	// 不允许添加自己
	if req.UserID == ownerID {
		return nil, apperrors.New(apperrors.CodeParamInvalid, "不能添加自己为成员")
	}

	// 验证目标用户存在
	user, err := s.userRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if user == nil {
		return nil, apperrors.CodeError(apperrors.CodeUserNotFound)
	}

	// 检查是否已是成员
	alreadyMember, err := s.repo.IsMember(ctx, projectID, req.UserID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if alreadyMember {
		return nil, apperrors.CodeError(apperrors.CodeMemberAlreadyExists)
	}

	member := &model.ProjectMember{
		ProjectID:    projectID,
		UserID:       req.UserID,
		Role:         model.ProjectMemberRole(req.Role),
		Contribution: req.Contribution,
		JoinedAt:     time.Now(),
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

// RemoveMember 移除项目成员（仅 owner）
// targetUserID 为被移除成员的 user_id
func (s *projectService) RemoveMember(ctx context.Context, ownerID, projectID, targetUserID uint64) error {
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}

	// 不允许移除自己（owner）
	if targetUserID == ownerID {
		return apperrors.New(apperrors.CodeParamInvalid, "不能移除自己")
	}

	// 查询被移除成员，确保其不是 owner 角色
	targetMember, err := s.repo.GetMemberByUserID(ctx, projectID, targetUserID)
	if err != nil {
		return err
	}

	if targetMember.Role == model.ProjectMemberRoleOwner {
		return apperrors.New(apperrors.CodeParamInvalid, "不能移除项目负责人")
	}

	if err := s.repo.RemoveMember(ctx, projectID, targetUserID); err != nil {
		return err
	}
	return nil
}

// owner 不可退出（owner 需先转让项目所有权）
func (s *projectService) LeaveProject(ctx context.Context, userID, projectID uint64) error {
	member, err := s.repo.GetMemberByUserID(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if member.Role == model.ProjectMemberRoleOwner {
		return apperrors.New(apperrors.CodeParamInvalid, "负责人不能直接退出项目，请先转让项目所有权")
	}
	if err := s.repo.LeaveProject(ctx, projectID, userID); err != nil {
		return err
	}
	return nil
}

// targetUserID 为目标成员的 user_id
func (s *projectService) UpdateMember(ctx context.Context, ownerID, projectID, targetUserID uint64, req *UpdateMemberReq) error {
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}

	if !model.ValidProjectMemberRoles[model.ProjectMemberRole(req.Role)] {
		return apperrors.New(apperrors.CodeParamInvalid, "无效的成员角色")
	}

	// 查询目标成员，确保存在且不是 owner
	targetMember, err := s.repo.GetMemberByUserID(ctx, projectID, targetUserID)
	if err != nil {
		return err
	}

	if targetMember.Role == model.ProjectMemberRoleOwner {
		return apperrors.New(apperrors.CodeParamInvalid, "不能修改项目负责人的角色")
	}

	return s.repo.UpdateMember(ctx, projectID, targetUserID, map[string]interface{}{
		"role":         req.Role,
		"contribution": req.Contribution,
	})
}

// ListProjects 查询项目列表（带分页筛选）
func (s *projectService) ListProjects(ctx context.Context, params ListProjectsParams) ([]*ProjectListItem, int64, error) {
	repoParams := &repository.ListProjectsParams{
		Keyword: params.Keyword,
		OwnerID: params.OwnerID,
		Offset:  (params.Page - 1) * params.PageSize,
		Limit:   params.PageSize,
		SortBy:  params.SortBy,
	}
	if params.Status != "" {
		repoParams.Status = model.ProjectStatus(params.Status)
	}
	if params.Genre != "" {
		repoParams.Genre = model.ProjectGenre(params.Genre)
	}

	projects, total, err := s.repo.ListProjects(ctx, repoParams)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*ProjectListItem, 0, len(projects))
	for _, p := range projects {
		item := &ProjectListItem{
			ID:            p.ID,
			OwnerID:       p.OwnerID,
			Name:          p.Name,
			Slug:          p.Slug,
			Description:   p.Description,
			Genre:         p.Genre,
			StyleTags:     parseStringSlice(p.StyleTags),
			Status:        p.Status,
			CoverURL:      s.getPublicURL(ctx, p.CoverKey),
			FollowerCount: p.FollowerCount,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		}
		items = append(items, item)
	}

	return items, total, nil
}

// GetTimeline 获取项目时间轴（分页）
func (s *projectService) GetTimeline(ctx context.Context, projectID uint64, page, pageSize int) ([]*model.ProjectTimeline, int64, error) {
	if _, err := s.mustGetProject(ctx, projectID); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	return s.repo.GetTimeline(ctx, projectID, offset, pageSize)
}

// GetProjectMembers 获取项目成员列表（公开接口）
func (s *projectService) GetProjectMembers(ctx context.Context, projectID uint64) ([]*model.ProjectMember, error) {
	if _, err := s.mustGetProject(ctx, projectID); err != nil {
		return nil, err
	}
	return s.repo.GetMembers(ctx, projectID)
}

func (s *projectService) requireProjectMember(ctx context.Context, projectID, userID uint64) error {
	project, err := s.mustGetProject(ctx, projectID)
	if err != nil {
		return err
	}
	if project.OwnerID == userID {
		return nil
	}
	isMember, err := s.repo.IsMember(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return apperrors.CodeError(apperrors.CodeProjectForbidden)
	}
	return nil
}

func (s *projectService) taskDetail(task *model.ProjectTask) *ProjectTaskDetail {
	detail := &ProjectTaskDetail{
		ID:          task.ID,
		ProjectID:   task.ProjectID,
		CreatorID:   task.CreatorID,
		AssigneeID:  task.AssigneeID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Priority:    task.Priority,
		DueDate:     task.DueDate,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
	if task.Creator.ID != 0 {
		detail.CreatorNickname = task.Creator.Nickname
	}
	if task.Assignee != nil && task.Assignee.ID != 0 {
		detail.AssigneeNickname = task.Assignee.Nickname
	}
	return detail
}

func (s *projectService) ListTasks(ctx context.Context, userID, projectID uint64) ([]*ProjectTaskDetail, error) {
	if err := s.requireProjectMember(ctx, projectID, userID); err != nil {
		return nil, err
	}
	tasks, err := s.repo.ListTasks(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]*ProjectTaskDetail, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, s.taskDetail(task))
	}
	return result, nil
}

func validateTaskReq(req *ProjectTaskReq, creating bool) (map[string]interface{}, error) {
	updates := map[string]interface{}{}
	if creating && (req.Title == nil || strings.TrimSpace(*req.Title) == "") {
		return nil, apperrors.New(apperrors.CodeParamMissing, "任务标题不能为空")
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" || len([]rune(title)) > 120 {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "任务标题长度需为 1-120")
		}
		updates["title"] = title
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		status := model.ProjectTaskStatus(*req.Status)
		if !model.ValidProjectTaskStatuses[status] {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "任务状态无效")
		}
		updates["status"] = status
	}
	if req.Priority != nil {
		priority := model.ProjectTaskPriority(*req.Priority)
		if !model.ValidProjectTaskPriorities[priority] {
			return nil, apperrors.New(apperrors.CodeParamInvalid, "任务优先级无效")
		}
		updates["priority"] = priority
	}
	if req.AssigneeID != nil {
		if *req.AssigneeID == 0 {
			updates["assignee_id"] = nil
		} else {
			updates["assignee_id"] = *req.AssigneeID
		}
	}
	if req.DueDate != nil {
		if strings.TrimSpace(*req.DueDate) == "" {
			updates["due_date"] = nil
		} else {
			parsed, err := time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				return nil, apperrors.New(apperrors.CodeParamInvalid, "截止日期格式应为 YYYY-MM-DD")
			}
			updates["due_date"] = parsed
		}
	}
	return updates, nil
}

func (s *projectService) CreateTask(ctx context.Context, userID, projectID uint64, req *ProjectTaskReq) (*ProjectTaskDetail, error) {
	if err := s.requireProjectMember(ctx, projectID, userID); err != nil {
		return nil, err
	}
	updates, err := validateTaskReq(req, true)
	if err != nil {
		return nil, err
	}
	task := &model.ProjectTask{ProjectID: projectID, CreatorID: userID, Status: model.ProjectTaskStatusTodo, Priority: model.ProjectTaskPriorityMedium}
	if v, ok := updates["title"].(string); ok {
		task.Title = v
	}
	if v, ok := updates["description"].(string); ok {
		task.Description = v
	}
	if v, ok := updates["status"].(model.ProjectTaskStatus); ok {
		task.Status = v
	}
	if v, ok := updates["priority"].(model.ProjectTaskPriority); ok {
		task.Priority = v
	}
	if v, ok := updates["assignee_id"].(uint64); ok {
		task.AssigneeID = &v
	}
	if v, ok := updates["due_date"].(time.Time); ok {
		task.DueDate = &v
	}
	if task.AssigneeID != nil {
		if err := s.requireProjectMember(ctx, projectID, *task.AssigneeID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	task, err = s.repo.GetTaskByID(ctx, projectID, task.ID)
	if err != nil {
		return nil, err
	}
	return s.taskDetail(task), nil
}

func (s *projectService) UpdateTask(ctx context.Context, userID, projectID, taskID uint64, req *ProjectTaskReq) (*ProjectTaskDetail, error) {
	if err := s.requireProjectMember(ctx, projectID, userID); err != nil {
		return nil, err
	}
	updates, err := validateTaskReq(req, false)
	if err != nil {
		return nil, err
	}
	if v, ok := updates["assignee_id"].(uint64); ok {
		if err := s.requireProjectMember(ctx, projectID, v); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateTask(ctx, projectID, taskID, updates); err != nil {
		return nil, err
	}
	task, err := s.repo.GetTaskByID(ctx, projectID, taskID)
	if err != nil {
		return nil, err
	}
	return s.taskDetail(task), nil
}

func (s *projectService) DeleteTask(ctx context.Context, userID, projectID, taskID uint64) error {
	isOwner, err := s.repo.IsOwner(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return apperrors.CodeError(apperrors.CodeProjectForbidden)
	}
	return s.repo.DeleteTask(ctx, projectID, taskID)
}

// indexProject 异步将项目写入 ES 索引（失败不影响主流程）
func (s *projectService) indexProject(ctx context.Context, p *model.Project) {
	if s.searcher == nil {
		return
	}
	var tags []string
	if p.StyleTags != "" {
		if err := json.Unmarshal([]byte(p.StyleTags), &tags); err != nil {
			logger.Warn("failed to unmarshal JSON", zap.Error(err))
		}
	}
	if err := s.searcher.IndexProject(ctx, search.ProjectDoc{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Genre:       string(p.Genre),
		Tags:        tags,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt.Unix(),
	}); err != nil {
		logger.Warn("failed to index project", zap.Error(err))
	}
}

// GetUserProjects 获取指定用户的公开项目列表（分页，含 owner + 成员身份的项目）
func (s *projectService) GetUserProjects(ctx context.Context, userID uint64, page, pageSize int) ([]*ProjectListItem, int64, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	projects, total, err := s.repo.ListUserProjects(ctx, user.ID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*ProjectListItem, 0, len(projects))
	for _, p := range projects {
		item := &ProjectListItem{
			ID:            p.ID,
			OwnerID:       p.OwnerID,
			Name:          p.Name,
			Slug:          p.Slug,
			Description:   p.Description,
			Genre:         p.Genre,
			StyleTags:     parseStringSlice(p.StyleTags),
			Status:        p.Status,
			CoverURL:      s.getPublicURL(ctx, p.CoverKey),
			FollowerCount: p.FollowerCount,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		}
		items = append(items, item)
	}
	return items, total, nil
}

// TransferOwner 转让项目所有权（仅当前 owner 可操作）
func (s *projectService) TransferOwner(ctx context.Context, ownerID, projectID, newOwnerID uint64) error {
	if err := s.requireOwner(ctx, projectID, ownerID); err != nil {
		return err
	}
	if ownerID == newOwnerID {
		return apperrors.New(apperrors.CodeParamInvalid, "不能转让给自己")
	}
	_, err := s.repo.GetMemberByUserID(ctx, projectID, newOwnerID)
	if err != nil {
		return apperrors.New(apperrors.CodeNotFound, "目标用户不是项目成员")
	}
	if err := s.repo.UpdateMemberRole(ctx, projectID, ownerID, model.ProjectMemberRoleMember); err != nil {
		return err
	}
	if err := s.repo.UpdateMemberRole(ctx, projectID, newOwnerID, model.ProjectMemberRoleOwner); err != nil {
		return err
	}
	return s.repo.UpdateOwner(ctx, projectID, newOwnerID)
}

// ===========================
// 项目收藏
// ===========================

// CollectProject 收藏项目
func (s *projectService) CollectProject(ctx context.Context, userID, projectID uint64) error {
	// 确认项目存在
	if _, err := s.mustGetProject(ctx, projectID); err != nil {
		return err
	}
	return s.repo.CollectProject(ctx, projectID, userID)
}

// UncollectProject 取消收藏
func (s *projectService) UncollectProject(ctx context.Context, userID, projectID uint64) error {
	return s.repo.UncollectProject(ctx, projectID, userID)
}

// GetMyCollectedProjects 获取我收藏的项目列表（分页）
func (s *projectService) GetMyCollectedProjects(ctx context.Context, userID uint64, page, pageSize int) ([]*ProjectListItem, int64, error) {
	offset := (page - 1) * pageSize
	projects, total, err := s.repo.GetUserCollectedProjects(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*ProjectListItem, 0, len(projects))
	for _, p := range projects {
		item := &ProjectListItem{
			ID:            p.ID,
			OwnerID:       p.OwnerID,
			Name:          p.Name,
			Slug:          p.Slug,
			Description:   p.Description,
			Genre:         p.Genre,
			StyleTags:     parseStringSlice(p.StyleTags),
			Status:        p.Status,
			CoverURL:      s.getPublicURL(ctx, p.CoverKey),
			FollowerCount: p.FollowerCount,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		}
		items = append(items, item)
	}
	return items, total, nil
}
