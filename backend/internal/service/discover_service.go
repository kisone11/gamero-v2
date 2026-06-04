// Package service 提供发现模块（搜索与推荐）的业务逻辑层。
// 本文件包含以下功能：
//   - 搜索（项目/用户/日志/帖子关键词搜索）
//   - 热门内容（热门项目/热门日志分页/热门帖子）
//   - 个性化推荐帖子
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/metrics"
	"github.com/gamero/gamero/pkg/search"
	"github.com/gamero/gamero/pkg/storage"
)

// discoverPresignExpiry 发现模块中预签名 URL 有效期（24小时）
const discoverPresignExpiry = 24 * time.Hour

// ===========================
// 响应 DTO 定义
// ===========================

// DiscoverProjectItem 发现模块项目列表项（含预签名封面图 URL）
type DiscoverProjectItem struct {
	ID            uint64              `json:"id"`
	OwnerID       uint64              `json:"owner_id"`
	Name          string              `json:"name"`
	Slug          string              `json:"slug"`
	Description   string              `json:"description"`
	Genre         model.ProjectGenre  `json:"genre"`
	StyleTags     []string            `json:"style_tags"`     // 已解析的风格标签列表
	Status        model.ProjectStatus `json:"status"`
	CoverURL      string              `json:"cover_url,omitempty"` // 预签名封面图 URL
	FollowerCount int                 `json:"follower_count"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// DiscoverUserItem 发现模块用户列表项（含预签名头像 URL）
type DiscoverUserItem struct {
	ID        uint64         `json:"id"`
	Username  string         `json:"username"`
	Nickname  string         `json:"nickname"`
	AvatarURL string         `json:"avatar_url,omitempty"` // 预签名头像 URL
	Bio       string         `json:"bio,omitempty"`
	Role      model.UserRole `json:"role"`
	CreatedAt time.Time      `json:"created_at"`
}

// DiscoverLogItem 发现模块日志列表项
type DiscoverLogItem struct {
	ID           uint64                 `json:"id"`
	ProjectID    uint64                 `json:"project_id"`
	AuthorID     uint64                 `json:"author_id"`
	Title        string                 `json:"title"`
	LogType      model.DevLogType       `json:"log_type"`
	Version      string                 `json:"version,omitempty"`
	Visibility   model.DevLogVisibility `json:"visibility"`
	Status       model.DevLogStatus     `json:"status"`
	ImageURLs    []string               `json:"image_urls"`
	VideoURLs    []string               `json:"video_urls"`
	CoverURL     string                 `json:"cover_url,omitempty"`
	LikeCount    int                    `json:"like_count"`
	ViewCount    int                    `json:"view_count"`
	CommentCount int                    `json:"comment_count"`
	CollectCount int                    `json:"collect_count"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// DiscoverPostItem 社区帖子搜索结果项
type DiscoverPostItem struct {
	ID           uint64    `json:"id"`
	AuthorID     uint64    `json:"author_id"`
	TopicIDs     string    `json:"topic_ids,omitempty"` // JSON 存储的话题 ID 列表
	Title        string    `json:"title"`
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	CollectCount int       `json:"collect_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// ===========================
// Service 接口定义
// ===========================

// DiscoverService 发现模块业务逻辑接口
type DiscoverService interface {
	// ===== 搜索 =====

	// SearchProjects 搜索项目，支持分页
	SearchProjects(ctx context.Context, keyword string, page, pageSize int) ([]*DiscoverProjectItem, int64, error)

	// SearchUsers 搜索用户，支持分页
	SearchUsers(ctx context.Context, keyword string, page, pageSize int) ([]*DiscoverUserItem, int64, error)

	// SearchLogs 搜索已发布公开日志，支持分页
	SearchLogs(ctx context.Context, keyword string, page, pageSize int, userID uint64) ([]*DiscoverLogItem, int64, error)

	// SearchPosts 搜索社区帖子，支持分页
	SearchPosts(ctx context.Context, keyword string, page, pageSize int) ([]*DiscoverPostItem, int64, error)

	// ===== 热门/最新内容（公开接口）=====

	// GetHotProjects 热门项目：按关注数排序（前 limit 条）
	// userID 用于去重已展示内容；未登录传 0
	GetHotProjects(ctx context.Context, limit int, userID uint64) ([]*DiscoverProjectItem, error)

	// GetHotLogsPaged 热门日志分页
	// logType 非空时只返回该类型的日志
	// userID 用于去重已展示内容；未登录传 0
	GetHotLogsPaged(ctx context.Context, page, pageSize int, logType string, userID uint64) ([]*DiscoverLogItem, int64, error)

	// GetHotPosts 热门帖子分页
	GetHotPosts(ctx context.Context, page, pageSize int) ([]*DiscoverPostItem, int64, error)

	// GetRecommendedPosts 个性化推荐帖子（五级降级：聚合互动CF → 社交CF → Topic画像 → 分层热门 → 全局热门兜底）
	GetRecommendedPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*DiscoverPostItem, int64, error)
}

// ===========================
// Service 实现
// ===========================

// discoverService DiscoverService 接口的具体实现
type discoverService struct {
	repo     repository.DiscoverRepository
	storage  *storage.Client
	searcher search.Searcher // ES 或 NoopSearcher，nil 时纯走 PG FTS
}

// NewDiscoverService 创建 discoverService 实例
func NewDiscoverService(repo repository.DiscoverRepository, storage *storage.Client, searcher search.Searcher) DiscoverService {
	return &discoverService{
		repo:     repo,
		storage:  storage,
		searcher: searcher,
	}
}

func (s *discoverService) presignURL(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	url, err := s.storage.PresignedGetURL(ctx, key, discoverPresignExpiry)
	if err != nil {
		return s.storage.GetPublicURL(key)
	}
	return url
}

// ===========================
// 搜索实现
// ===========================

// SearchProjects 搜索项目，优先走 ES，降级 PG FTS
func (s *discoverService) SearchProjects(ctx context.Context, keyword string, page, pageSize int) ([]*DiscoverProjectItem, int64, error) {
	offset := (page - 1) * pageSize

	// 尝试 ES 搜索
	if s.searcher != nil {
		esResult, err := s.searcher.SearchProjects(ctx, keyword, offset, pageSize)
		if err == nil && len(esResult.Hits) > 0 {
			ids := make([]uint64, 0, len(esResult.Hits))
			for _, h := range esResult.Hits {
				ids = append(ids, h.ID)
			}
			projects, err := s.repo.GetProjectsByIDs(ctx, ids)
			if err == nil {
				items, err := s.convertProjects(ctx, projects)
				if err == nil {
					metrics.SearchQueryTotal.WithLabelValues("es", "project").Inc()
					return items, esResult.Total, nil
				}
			}
		}
	}

	// 降级：PostgreSQL FTS
	projects, total, err := s.repo.SearchProjects(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.convertProjects(ctx, projects)
	if err != nil {
		return nil, 0, err
	}
	metrics.SearchQueryTotal.WithLabelValues("pg_fts", "project").Inc()
	return items, total, nil
}

// SearchUsers 搜索用户，优先走 ES，降级 PG FTS
func (s *discoverService) SearchUsers(ctx context.Context, keyword string, page, pageSize int) ([]*DiscoverUserItem, int64, error) {
	offset := (page - 1) * pageSize

	if s.searcher != nil {
		esResult, err := s.searcher.SearchUsers(ctx, keyword, offset, pageSize)
		if err == nil && len(esResult.Hits) > 0 {
			ids := make([]uint64, 0, len(esResult.Hits))
			for _, h := range esResult.Hits {
				ids = append(ids, h.ID)
			}
			users, err := s.repo.GetUsersByIDs(ctx, ids)
			if err == nil {
				metrics.SearchQueryTotal.WithLabelValues("es", "user").Inc()
				return s.convertUsers(ctx, users), esResult.Total, nil
			}
		}
	}

	users, total, err := s.repo.SearchUsers(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	metrics.SearchQueryTotal.WithLabelValues("pg_fts", "user").Inc()
	return s.convertUsers(ctx, users), total, nil
}

// SearchLogs 搜索日志，优先走 ES，降级 PG FTS；keyword 为空时直接返回热门日志列表
func (s *discoverService) SearchLogs(ctx context.Context, keyword string, page, pageSize int, userID uint64) ([]*DiscoverLogItem, int64, error) {
	if keyword == "" {
		return s.GetHotLogsPaged(ctx, page, pageSize, "", userID)
	}
	offset := (page - 1) * pageSize

	if s.searcher != nil {
		esResult, err := s.searcher.SearchLogs(ctx, keyword, offset, pageSize)
		if err == nil && len(esResult.Hits) > 0 {
			ids := make([]uint64, 0, len(esResult.Hits))
			for _, h := range esResult.Hits {
				ids = append(ids, h.ID)
			}
			logs, err := s.repo.GetLogsByIDs(ctx, ids)
			if err == nil {
				metrics.SearchQueryTotal.WithLabelValues("es", "log").Inc()
				return s.convertLogs(ctx, logs), esResult.Total, nil
			}
		}
	}

	logs, total, err := s.repo.SearchLogs(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	metrics.SearchQueryTotal.WithLabelValues("pg_fts", "log").Inc()
	return s.convertLogs(ctx, logs), total, nil
}

// SearchPosts 搜索社区帖子，支持分页
func (s *discoverService) SearchPosts(ctx context.Context, keyword string, page, pageSize int) ([]*DiscoverPostItem, int64, error) {
	offset := (page - 1) * pageSize
	posts, total, err := s.repo.SearchPosts(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return convertPosts(posts), total, nil
}

// ===========================
// 热门内容实现
// ===========================

// GetHotProjects 热门项目（含 HotScore 排序 + 去重）
func (s *discoverService) GetHotProjects(ctx context.Context, limit int, userID uint64) ([]*DiscoverProjectItem, error) {
	// Fetch more items to allow dedup compensation
	projects, _, err := s.repo.GetProjectsByFollowerCount(ctx, 0, limit*3)
	if err != nil {
		return nil, err
	}

	// Apply dedup
	var filtered []*model.Project
	for _, p := range projects {
		if s.shouldInclude(ctx, userID, "project", p.ID) {
			filtered = append(filtered, p)
		}
	}

	// HotScore re-ranking (follower_count as engagement proxy)
	sort.Slice(filtered, func(i, j int) bool {
		return HotScore(int64(filtered[i].FollowerCount), 0, 0, filtered[i].CreatedAt) >
			HotScore(int64(filtered[j].FollowerCount), 0, 0, filtered[j].CreatedAt)
	})

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	items, err := s.convertProjects(ctx, filtered)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// ===========================
// 内部工具方法：数据转换
// ===========================

// convertProjects 将 []*model.Project 转换为 []*DiscoverProjectItem，
// 并为每个项目的封面图生成公开 URL
// 若 storage 调用失败，封面 URL 置空（不影响主流程）
func (s *discoverService) convertProjects(ctx context.Context, projects []*model.Project) ([]*DiscoverProjectItem, error) {
	items := make([]*DiscoverProjectItem, 0, len(projects))
	for _, p := range projects {
		item := &DiscoverProjectItem{
			ID:            p.ID,
			OwnerID:       p.OwnerID,
			Name:          p.Name,
			Slug:          p.Slug,
			Description:   p.Description,
			Genre:         p.Genre,
			StyleTags:     parseDiscoverStringSlice(p.StyleTags),
			Status:        p.Status,
			FollowerCount: p.FollowerCount,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		}

		// 生成封面图公开 URL
		if p.CoverKey != "" {
			item.CoverURL = s.presignURL(ctx, p.CoverKey)
		}

		items = append(items, item)
	}

	return items, nil
}

// convertUsers 将 []*model.User 转换为 []*DiscoverUserItem，
// 并为每个用户的头像生成公开 URL
func (s *discoverService) convertUsers(ctx context.Context, users []*model.User) []*DiscoverUserItem {
	items := make([]*DiscoverUserItem, 0, len(users))
	for _, u := range users {
		item := &DiscoverUserItem{
			ID:        u.ID,
			Username:  u.Username,
			Nickname:  u.Nickname,
			Bio:       u.Bio,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
		}

		// 生成头像公开 URL
		if u.AvatarKey != "" {
			item.AvatarURL = s.presignURL(ctx, u.AvatarKey)
		}

		items = append(items, item)
	}

	return items
}

// convertLogs 将 []*model.DevLog 转换为 []*DiscoverLogItem
func (s *discoverService) convertLogs(ctx context.Context, logs []*model.DevLog) []*DiscoverLogItem {
	items := make([]*DiscoverLogItem, 0, len(logs))
	for _, l := range logs {
		item := &DiscoverLogItem{
			ID:           l.ID,
			ProjectID:    l.ProjectID,
			AuthorID:     l.AuthorID,
			Title:        l.Title,
			LogType:      l.LogType,
			Version:      l.Version,
			Visibility:   l.Visibility,
			Status:       l.Status,
			ImageURLs:    s.presignKeys(ctx, parseImageKeys(l.Images)),
			VideoURLs:    s.presignKeys(ctx, parseImageKeys(l.Videos)),
			LikeCount:    l.LikeCount,
			ViewCount:    l.ViewCount,
			CommentCount: l.CommentCount,
			CollectCount: l.CollectCount,
			CreatedAt:    l.CreatedAt,
			UpdatedAt:    l.UpdatedAt,
		}
		items = append(items, item)
	}

	return items
}

// presignKeys 将存储 key 列表转换为预签名 URL 列表
func (s *discoverService) presignKeys(ctx context.Context, keys []string) []string {
	if len(keys) == 0 {
		return nil
	}
	urls := make([]string, 0, len(keys))
	for _, key := range keys {
		url, err := s.storage.PresignedGetURL(ctx, key, 24*time.Hour)
		if err != nil {
			urls = append(urls, s.storage.GetPublicURL(key))
			continue
		}
		urls = append(urls, url)
	}
	return urls
}

// convertPosts 将 model.Post 列表转换为 DiscoverPostItem 列表
func convertPosts(posts []*model.Post) []*DiscoverPostItem {
	items := make([]*DiscoverPostItem, 0, len(posts))
	for _, p := range posts {
		items = append(items, &DiscoverPostItem{
			ID:           p.ID,
			AuthorID:     p.AuthorID,
			TopicIDs:     p.TopicIDs,
			Title:        p.Title,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CollectCount: p.CollectCount,
			CreatedAt:    p.CreatedAt,
		})
	}
	return items
}

// parseDiscoverStringSlice 解析 JSON 字符串为 []string，供发现模块内部使用
// 避免与 project_service.go 中的同名函数冲突
func parseDiscoverStringSlice(jsonStr string) []string {
	var result []string
	if jsonStr == "" || jsonStr == "null" {
		return []string{}
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return []string{}
	}
	return result
}

// ===========================
// findIDInSlice — 工具函数
// ===========================

func findIDInSlice(ids []uint64, target uint64) (int, bool) {
	for i, id := range ids {
		if id == target {
			return i, true
		}
	}
	return -1, false
}

// ===========================
// GetHotLogsPaged — 热门日志分页（含 HotScore 排序 + 去重）
// ===========================

// GetHotLogsPaged 热门日志分页
// logType 非空时只返回该类型的日志
func (s *discoverService) GetHotLogsPaged(ctx context.Context, page, pageSize int, logType string, userID uint64) ([]*DiscoverLogItem, int64, error) {
	// Fetch more items to allow dedup compensation
	offset := (page - 1) * pageSize
	fetchSize := pageSize * 3
	logs, total, err := s.repo.GetTopLogs(ctx, offset, fetchSize, logType)
	if err != nil {
		return nil, 0, err
	}

	// Apply dedup
	var filtered []*model.DevLog
	for _, l := range logs {
		if s.shouldInclude(ctx, userID, "log", l.ID) {
			filtered = append(filtered, l)
		}
	}

	// HotScore re-ranking
	sort.Slice(filtered, func(i, j int) bool {
		return HotScore(int64(filtered[i].LikeCount), int64(filtered[i].CommentCount), int64(filtered[i].ViewCount), filtered[i].CreatedAt) >
			HotScore(int64(filtered[j].LikeCount), int64(filtered[j].CommentCount), int64(filtered[j].ViewCount), filtered[j].CreatedAt)
	})

	// Take top pageSize
	if len(filtered) > pageSize {
		filtered = filtered[:pageSize]
	}

	items := s.convertLogs(ctx, filtered)
	return items, total, nil
}

// ===========================
// GetHotPosts — 热门帖子分页（含 HotScore 排序）
// ===========================

func (s *discoverService) GetHotPosts(ctx context.Context, page, pageSize int) ([]*DiscoverPostItem, int64, error) {
	// Fetch more items for HotScore re-ranking
	fetchSize := pageSize * 3
	posts, total, err := s.repo.GetTopPosts(ctx, (page-1)*pageSize, fetchSize)
	if err != nil {
		return nil, 0, err
	}

	// HotScore re-ranking
	sort.Slice(posts, func(i, j int) bool {
		return HotScore(int64(posts[i].LikeCount), int64(posts[i].CommentCount), int64(posts[i].ViewCount), posts[i].CreatedAt) >
			HotScore(int64(posts[j].LikeCount), int64(posts[j].CommentCount), int64(posts[j].ViewCount), posts[j].CreatedAt)
	})

	if len(posts) > pageSize {
		posts = posts[:pageSize]
	}

	items := convertPosts(posts)
	return items, total, nil
}

// ===========================
// GetRecommendedPosts — 个性化帖子推荐（五级降级，30min 缓存）
// ===========================

func (s *discoverService) GetRecommendedPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*DiscoverPostItem, int64, error) {
	cacheKey := cache.RecommendPostsKey(userID)
	var recommendedIDs []uint64

	if _, err := cache.GetJSON(ctx, cacheKey, &recommendedIDs); err != nil {
		recommendedIDs = nil
	}
	if len(recommendedIDs) == 0 {
		ids, err := s.computePostRecommendations(ctx, userID)
		if err != nil || len(ids) == 0 {
			ids, _ = s.repo.GetHotPostIDs(ctx, nil, 50)
		}
		recommendedIDs = ids
		if err := cache.SetJSON(ctx, cacheKey, recommendedIDs, cache.TTLRecommend); err != nil {
			logger.Warn("failed to set cache", zap.Error(err))
		}
	}

	// Apply dedup: filter already-seen items
	var dedupedIDs []uint64
	for _, id := range recommendedIDs {
		if s.shouldInclude(ctx, userID, "post", id) {
			dedupedIDs = append(dedupedIDs, id)
		}
	}

	// If dedup removed too many, fetch more hot posts to compensate
	shortage := pageSize - len(dedupedIDs) + (page-1)*pageSize
	if shortage > 20 && len(dedupedIDs) < pageSize {
		extraIDs, _ := s.repo.GetHotPostIDs(ctx, dedupedIDs, shortage+10)
		for _, id := range extraIDs {
			if s.shouldInclude(ctx, userID, "post", id) {
				dedupedIDs = append(dedupedIDs, id)
			}
		}
	}

	recommendedIDs = dedupedIDs

	total := int64(len(recommendedIDs))
	offset := (page - 1) * pageSize
	if offset >= len(recommendedIDs) {
		return []*DiscoverPostItem{}, total, nil
	}
	end := offset + pageSize
	if end > len(recommendedIDs) {
		end = len(recommendedIDs)
	}
	pageIDs := recommendedIDs[offset:end]

	posts, err := s.repo.GetDiscoverPostsByIDs(ctx, pageIDs)
	if err != nil {
		return nil, 0, err
	}
	postMap := make(map[uint64]*model.Post, len(posts))
	for _, p := range posts {
		postMap[p.ID] = p
	}
	ordered := make([]*model.Post, 0, len(pageIDs))
	for _, id := range pageIDs {
		if p, ok := postMap[id]; ok {
			ordered = append(ordered, p)
		}
	}

	// HotScore re-ranking of displayed posts
	sort.Slice(ordered, func(i, j int) bool {
		return HotScore(int64(ordered[i].LikeCount), int64(ordered[i].CommentCount), int64(ordered[i].ViewCount), ordered[i].CreatedAt) >
			HotScore(int64(ordered[j].LikeCount), int64(ordered[j].CommentCount), int64(ordered[j].ViewCount), ordered[j].CreatedAt)
	})

	items := convertPosts(ordered)

	return items, total, nil
}

// computePostRecommendations 帖子推荐五级降级（每级均进行 HotScore 重排）：
// L1 聚合互动 Item-CF → L2 社交 CF → L3 Topic画像 → L4 分层热门 → L5 全局热门
func (s *discoverService) computePostRecommendations(ctx context.Context, userID uint64) ([]uint64, error) {
	// L1: 聚合互动 Item-CF（点赞+收藏+评论）
	interactedIDs, err := s.repo.GetUserInteractedPostIDs(ctx, userID, 30)
	if err == nil && len(interactedIDs) > 0 {
		coItems, coErr := s.repo.GetCoInteractedPostIDs(ctx, interactedIDs, interactedIDs, 50)
		if coErr == nil && len(coItems) > 0 {
			return s.rankPostsByHotScore(ctx, coItems)
		}
	}
	// L2: 社交 CF（我关注的人互动过的帖子）
	socialIDs, socialErr := s.repo.GetSocialFeedPostIDs(ctx, userID, interactedIDs, 50)
	if socialErr == nil && len(socialIDs) > 0 {
		return s.rankPostsByHotScore(ctx, socialIDs)
	}
	// L3: 话题画像（从用户帖子历史 + 互动帖子推断偏好话题）
	topicIDs, _ := s.repo.GetUserPreferredTopicIDs(ctx, userID, 5)
	if len(topicIDs) > 0 {
		topicHotIDs, tErr := s.repo.GetHotPostsByTopics(ctx, topicIDs, interactedIDs, 50)
		if tErr == nil && len(topicHotIDs) > 0 {
			return s.rankPostsByHotScore(ctx, topicHotIDs)
		}
	}
	// L4: 按话题查帖子（L3 的宽松版，不限热度）
	if len(topicIDs) > 0 {
		posts, _, pErr := s.repo.GetPostsByTopicIDs(ctx, topicIDs, interactedIDs, 0, 50)
		if pErr == nil && len(posts) > 0 {
			sort.Slice(posts, func(i, j int) bool {
				return HotScore(int64(posts[i].LikeCount), int64(posts[i].CommentCount), int64(posts[i].ViewCount), posts[i].CreatedAt) >
					HotScore(int64(posts[j].LikeCount), int64(posts[j].CommentCount), int64(posts[j].ViewCount), posts[j].CreatedAt)
			})
			ids := make([]uint64, 0, len(posts))
			for _, p := range posts {
				ids = append(ids, p.ID)
			}
			return ids, nil
		}
	}
	// L5: 全局热门（最后兜底）
	ids, _ := s.repo.GetHotPostIDs(ctx, interactedIDs, 50)
	return s.rankPostsByHotScore(ctx, ids)
}

// ===========================
// Redis 去重辅助方法
// ===========================

// shouldInclude 检查 itemID 是否未被用户看过（Redis SADD 去重）。
// userID=0 时跳过检查（未登录用户不做去重）。
// 返回 true 表示应展示该内容；返回 false 表示用户已看过应跳过。
func (s *discoverService) shouldInclude(ctx context.Context, userID uint64, contentType string, itemID uint64) bool {
	if userID == 0 {
		return true // 未登录，不做去重
	}
	rdb := cache.Get()
	if rdb == nil {
		return true // Redis 未初始化，放行
	}
	key := fmt.Sprintf("recommend:seen:%d:%s", userID, contentType)
	member := fmt.Sprintf("%d", itemID)
	added, err := rdb.SAdd(ctx, key, member).Result()
	if err != nil {
		return true // Redis 错误，放行
	}
	// 设置 TTL（每次 SAdd 后刷新，确保 key 持续存活）
	rdb.Expire(ctx, key, 24*time.Hour)
	return added > 0 // >0 表示首次加入（未看过），应展示
}

// ===========================
// HotScore 重排辅助方法
// ===========================

// rankPostsByHotScore 根据 HotScore 对帖子 ID 列表降序重排。
// 需要回查 DB 获取帖子元数据以计算热度分数。
func (s *discoverService) rankPostsByHotScore(ctx context.Context, ids []uint64) ([]uint64, error) {
	if len(ids) == 0 {
		return ids, nil
	}
	posts, err := s.repo.GetDiscoverPostsByIDs(ctx, ids)
	if err != nil || len(posts) == 0 {
		return ids, nil // 回查失败，保持原顺序
	}
	postMap := make(map[uint64]*model.Post, len(posts))
	for _, p := range posts {
		postMap[p.ID] = p
	}
	sort.Slice(ids, func(i, j int) bool {
		pi, okI := postMap[ids[i]]
		pj, okJ := postMap[ids[j]]
		if !okI {
			return false
		}
		if !okJ {
			return true
		}
		return HotScore(int64(pi.LikeCount), int64(pi.CommentCount), int64(pi.ViewCount), pi.CreatedAt) >
			HotScore(int64(pj.LikeCount), int64(pj.CommentCount), int64(pj.ViewCount), pj.CreatedAt)
	})
	return ids, nil
}
