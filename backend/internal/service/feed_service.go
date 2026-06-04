// Package service 提供统一的关注内容动态流业务逻辑。
//
// Feed 算法：
//   - 关注用户的帖子（type=post）
//   - 关注用户的公开已发布开发日志（type=devlog）
//   - 关注用户创建的项目（type=project）
//   - 关注项目的版本发布日志（type=release）
//
// 采用游标分页，所有来源按 created_at 倒序归并后返回。
package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/storage"
)

// ===========================
// Feed 响应 DTO
// ===========================

// FeedItemResponse 统一动态流中的单条记录
type FeedItemResponse struct {
	ID        string             `json:"id"`                 // "{type}_{id}" 格式的复合 ID
	Type      string             `json:"type"`               // "post" | "devlog" | "project" | "release"
	Author    *FeedAuthor        `json:"author"`             // 作者信息
	Post      *FeedPostDetail    `json:"post,omitempty"`     // type=post 时填充
	DevLog    *FeedDevLogDetail  `json:"devlog,omitempty"`   // type=devlog / release 时填充
	Project   *FeedProjectDetail `json:"project,omitempty"`  // type=project 时填充
	CreatedAt time.Time          `json:"created_at"`
}

// FeedAuthor 作者简要信息
type FeedAuthor struct {
	ID        uint64 `json:"id"`
	Nickname  string `json:"nickname"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

// FeedPostDetail 帖子的详细字段
type FeedPostDetail struct {
	ID             uint64   `json:"id"`
	Title          string   `json:"title"`
	ContentSnippet string   `json:"content_snippet"` // 纯文本摘要（strip markdown）
	ImageURLs      []string `json:"image_urls"`
	LikeCount      int      `json:"like_count"`
	CommentCount   int      `json:"comment_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// FeedDevLogDetail 开发日志/版本发布的详细字段
type FeedDevLogDetail struct {
	ID        uint64   `json:"id"`
	ProjectID uint64   `json:"project_id"`
	Title     string   `json:"title"`
	LogType   string   `json:"log_type"` // "log" | "release"
	ImageURLs []string `json:"image_urls"`
	CreatedAt time.Time `json:"created_at"`
}

// FeedProjectDetail 项目的详细字段
type FeedProjectDetail struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CoverURL    string `json:"cover_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ===========================
// FeedService 接口
// ===========================

// FeedService 动态流服务接口
type FeedService interface {
	// GetFeed 获取当前用户的统一动态流（游标分页）。
	// cursor: 上一页最后一条的 created_at UnixNano 时间戳，0 表示从最新开始。
	// limit: 每页条数（默认 20，最大 100）。
	// 返回 items 和 nextCursor（0 表示已到末尾）。
	GetFeed(ctx context.Context, userID uint64, cursor int64, limit int) ([]*FeedItemResponse, int64, error)
}

// ===========================
// feedService 实现
// ===========================

type feedService struct {
	followRepo  repository.FollowRepository
	userRepo    repository.UserRepository
	projectRepo repository.ProjectRepository
	devLogRepo  repository.DevLogRepository
	postRepo    repository.PostRepository
	storage     *storage.Client
}

// NewFeedService 创建 feedService 实例
func NewFeedService(
	followRepo repository.FollowRepository,
	userRepo repository.UserRepository,
	projectRepo repository.ProjectRepository,
	devLogRepo repository.DevLogRepository,
	postRepo repository.PostRepository,
	storageClient *storage.Client,
) FeedService {
	return &feedService{
		followRepo:  followRepo,
		userRepo:    userRepo,
		projectRepo: projectRepo,
		devLogRepo:  devLogRepo,
		postRepo:    postRepo,
		storage:     storageClient,
	}
}

// feedConstants 查询限制
const (
	feedMaxFetchPerType = 200 // 每类最多拉取条数
)

// internalFeedItem 内部归并排序用的中间结构
type internalFeedItem struct {
	itemType  string    // "post" | "devlog" | "project" | "release"
	authorID  uint64    // 作者/owner ID
	createdAt time.Time
	// 具体数据指针（只填充一个）
	post    *model.Post
	devLog  *model.DevLog
	project *model.Project
}

// compositeID 生成 {type}_{id} 格式的复合 ID
func compositeID(itemType string, id uint64) string {
	return fmt.Sprintf("%s_%d", itemType, id)
}

// GetFeed 获取当前用户的统一动态流
func (s *feedService) GetFeed(ctx context.Context, userID uint64, cursor int64, limit int) ([]*FeedItemResponse, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if cursor < 0 {
		cursor = 0
	}

	// 1. 获取关注的用户 ID 和项目 ID
	followingUserIDs, err := s.followRepo.GetFollowingIDs(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("get following user ids: %w", err)
	}
	followingProjectIDs, err := s.followRepo.GetFollowingProjectIDs(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("get following project ids: %w", err)
	}

	// 加入自己的内容：将当前用户加入关注用户列表
	followingUserIDs = append(followingUserIDs, userID)

	// 加入自己的项目：查询当前用户创建的项目 ID，加入关注项目列表
	ownProjectIDs, err := s.projectRepo.GetProjectIDsByOwnerID(ctx, userID)
	if err == nil && len(ownProjectIDs) > 0 {
		followingProjectIDs = append(followingProjectIDs, ownProjectIDs...)
	}

	hasUsers := len(followingUserIDs) > 0
	hasProjects := len(followingProjectIDs) > 0

	if !hasUsers && !hasProjects {
		return []*FeedItemResponse{}, 0, nil
	}

	// 2. 批量拉取四类动态数据
	allItems := make([]internalFeedItem, 0, feedMaxFetchPerType*4)

	// 2a. 关注用户的帖子
	if hasUsers {
		posts, err := s.postRepo.ListPublicPostsByAuthorIDs(ctx, followingUserIDs, feedMaxFetchPerType)
		if err != nil {
			return nil, 0, fmt.Errorf("list posts by author ids: %w", err)
		}
		for _, p := range posts {
			allItems = append(allItems, internalFeedItem{
				itemType:  "post",
				authorID:  p.AuthorID,
				createdAt: p.CreatedAt,
				post:      p,
			})
		}
	}

	// 2b. 关注用户的公开已发布日志
	if hasUsers {
		logs, err := s.devLogRepo.ListPublicLogsByAuthorIDs(ctx, followingUserIDs, feedMaxFetchPerType)
		if err != nil {
			return nil, 0, fmt.Errorf("list devlogs by author ids: %w", err)
		}
		for _, dl := range logs {
			allItems = append(allItems, internalFeedItem{
				itemType:  "devlog",
				authorID:  dl.AuthorID,
				createdAt: dl.CreatedAt,
				devLog:    dl,
			})
		}
	}

	// 2c. 关注用户创建的项目（状态为 developing / playable / launched）
	if hasUsers {
		projects, err := s.projectRepo.ListProjectsByOwnerIDs(ctx, followingUserIDs, []model.ProjectStatus{
			model.ProjectStatusDeveloping,
			model.ProjectStatusPlayable,
			model.ProjectStatusLaunched,
		}, feedMaxFetchPerType)
		if err != nil {
			return nil, 0, fmt.Errorf("list projects by owner ids: %w", err)
		}
		for _, pr := range projects {
			allItems = append(allItems, internalFeedItem{
				itemType:  "project",
				authorID:  pr.OwnerID,
				createdAt: pr.CreatedAt,
				project:   pr,
			})
		}
	}

	// 2d. 关注项目的开发日志（含发布和普通日志）
	if hasProjects {
		projectLogs, err := s.devLogRepo.ListPublicLogsByProjectIDs(ctx, followingProjectIDs, feedMaxFetchPerType)
		if err != nil {
			return nil, 0, fmt.Errorf("list logs by project ids: %w", err)
		}
		for _, dl := range projectLogs {
			itemType := "devlog"
			if dl.LogType == model.DevLogTypeRelease {
				itemType = "release"
			}
			allItems = append(allItems, internalFeedItem{
				itemType:  itemType,
				authorID:  dl.AuthorID,
				createdAt: dl.CreatedAt,
				devLog:    dl,
			})
		}
	}

	if len(allItems) == 0 {
		return []*FeedItemResponse{}, 0, nil
	}

	// 3. 按 created_at 倒序排序
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].createdAt.After(allItems[j].createdAt)
	})

	// 4. 游标分页
	var cursorTime time.Time
	if cursor > 0 {
		cursorTime = time.Unix(0, cursor)
	}

	startIdx := 0
	if cursor > 0 {
		// 找到第一个 created_at < cursorTime 的条目
		startIdx = sort.Search(len(allItems), func(i int) bool {
			return allItems[i].createdAt.Before(cursorTime)
		})
	}

	if startIdx >= len(allItems) {
		return []*FeedItemResponse{}, 0, nil
	}

	endIdx := startIdx + limit
	if endIdx > len(allItems) {
		endIdx = len(allItems)
	}

	pageItems := allItems[startIdx:endIdx]

	// 5. 计算 next_cursor
	var nextCursor int64
	if endIdx < len(allItems) {
		nextCursor = allItems[endIdx-1].createdAt.UnixNano()
	}

	// 6. 批量查询页面内所有作者
	authorIDs := make(map[uint64]bool)
	for _, item := range pageItems {
		if item.authorID > 0 {
			authorIDs[item.authorID] = true
		}
	}
	userMap := make(map[uint64]*model.User)
	if len(authorIDs) > 0 {
		ids := make([]uint64, 0, len(authorIDs))
		for id := range authorIDs {
			ids = append(ids, id)
		}
		users, err := s.userRepo.GetUsersByIDs(ctx, ids)
		if err == nil {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	// 7. 转换为响应 DTO
	responses := make([]*FeedItemResponse, 0, len(pageItems))
	for _, item := range pageItems {
		resp := &FeedItemResponse{
			ID:        compositeID(item.itemType, itemID(item)),
			Type:      item.itemType,
			CreatedAt: item.createdAt,
		}

		// 作者信息
		if u, ok := userMap[item.authorID]; ok {
			avatarURL := ""
			if u.AvatarKey != "" {
				avatarURL = s.presignURL(ctx, u.AvatarKey)
			}
			resp.Author = &FeedAuthor{
				ID:        u.ID,
				Nickname:  u.Nickname,
				Username:  u.Username,
				AvatarURL: avatarURL,
			}
		}

		switch item.itemType {
		case "post":
			if item.post != nil {
				resp.Post = &FeedPostDetail{
					ID:             item.post.ID,
					Title:          item.post.Title,
					ContentSnippet: stripMarkdownAndTruncate(item.post.Content, 200),
					ImageURLs:      s.buildImageURLs(ctx, item.post.Images),
					LikeCount:      item.post.LikeCount,
					CommentCount:   item.post.CommentCount,
					CreatedAt:      item.post.CreatedAt,
				}
			}
		case "devlog", "release":
			if item.devLog != nil {
				resp.DevLog = &FeedDevLogDetail{
					ID:        item.devLog.ID,
					ProjectID: item.devLog.ProjectID,
					Title:     item.devLog.Title,
					LogType:   string(item.devLog.LogType),
					ImageURLs: s.buildImageURLs(ctx, item.devLog.Images),
					CreatedAt: item.devLog.CreatedAt,
				}
			}
		case "project":
			if item.project != nil {
				coverURL := ""
				if item.project.CoverKey != "" {
					coverURL = s.presignURL(ctx, item.project.CoverKey)
				}
				resp.Project = &FeedProjectDetail{
					ID:          item.project.ID,
					Name:        item.project.Name,
					Description: item.project.Description,
					Status:      string(item.project.Status),
					CoverURL:    coverURL,
					CreatedAt:   item.project.CreatedAt,
				}
			}
		}

		responses = append(responses, resp)
	}

	return responses, nextCursor, nil
}

// ===========================
// 内部工具方法
// ===========================

// itemID 提取内部 feed item 的数值 ID
func itemID(item internalFeedItem) uint64 {
	switch {
	case item.post != nil:
		return item.post.ID
	case item.devLog != nil:
		return item.devLog.ID
	case item.project != nil:
		return item.project.ID
	default:
		return 0
	}
}

// buildImageURLs 将对象存储 key 列表的 JSON 字符串转换为预签名 URL 列表
func (s *feedService) buildImageURLs(ctx context.Context, imagesJSON string) []string {
	keys := parseImageKeys(imagesJSON)
	urls := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		url, err := s.storage.PresignedGetURL(ctx, key, 24*time.Hour)
		if err != nil {
			url = s.storage.GetPublicURL(key)
		}
		urls = append(urls, url)
	}
	return urls
}

// presignURL 为单个 key 生成预签名公开 URL
func (s *feedService) presignURL(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	url, err := s.storage.PresignedGetURL(ctx, key, 24*time.Hour)
	if err != nil {
		return s.storage.GetPublicURL(key)
	}
	return url
}

// stripMarkdownAndTruncate 剥离 Markdown 标记并截断到指定长度
func stripMarkdownAndTruncate(md string, maxLen int) string {
	// 移除图片 ![alt](url)
	re := regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	plain := re.ReplaceAllString(md, "")

	// 移除链接 [text](url) -> text
	re = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	plain = re.ReplaceAllString(plain, "$1")

	// 移除代码块 ```...```
	re = regexp.MustCompile("(?s)```.*?```")
	plain = re.ReplaceAllString(plain, "")

	// 移除行内代码 `code`
	re = regexp.MustCompile("`[^`]*`")
	plain = re.ReplaceAllString(plain, "")

	// 移除标题标记 #
	re = regexp.MustCompile(`^#+\s*`)
	plain = re.ReplaceAllString(plain, "")

	// 移除粗体/斜体标记
	re = regexp.MustCompile(`[*_~]{1,3}`)
	plain = re.ReplaceAllString(plain, "")

	// 移除 HTML 标签
	re = regexp.MustCompile(`<[^>]*>`)
	plain = re.ReplaceAllString(plain, "")

	// 移除空行和多余空白
	plain = strings.TrimSpace(plain)
	plain = strings.Join(strings.Fields(plain), " ")

	// 截断
	runes := []rune(plain)
	if len(runes) <= maxLen {
		return plain
	}
	// 在单词边界截断
	truncated := string(runes[:maxLen])
	// 确保不在单词中间截断
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}

// stripAllNonPrintable 保留可打印字符（备用函数）
func stripAllNonPrintable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
}
