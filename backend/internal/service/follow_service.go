// Package service 提供关注与通知系统的业务逻辑层。
// 本文件包含以下业务逻辑：
//   - 用户关注/取关（含事务更新双方计数）
//   - 项目关注/取关（含关注数更新）
//   - 通知列表查询、标记已读
//   - 动态流（关注用户的日志 + 关注项目的时间轴事件，内存归并排序）
package service

import (
	"go.uber.org/zap"
	"github.com/gamero/gamero/pkg/logger"
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	apperrors "github.com/gamero/gamero/pkg/errors"
	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/metrics"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ===========================
// 请求/响应数据结构
// ===========================

// UserBriefInfo 用户基本信息（用于粉丝/关注列表展示）
type UserBriefInfo struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarKey string `json:"avatar_key,omitempty"`
	Bio       string `json:"bio,omitempty"`
}

// FollowUserItem 粉丝/关注列表中的一条用户记录
type FollowUserItem struct {
	UserBriefInfo
	AvatarURL      string        `json:"avatar_url"`      // 预签名公开 URL
	FollowersCount int           `json:"followers_count"` // 粉丝数
	FollowingCount int           `json:"following_count"` // 关注数
	IsFollowing    bool          `json:"is_following"`    // viewer 是否关注了该用户（先填 false）
	Projects       []interface{} `json:"projects"`        // 简化项目列表（空数组）
	FollowedAt     time.Time     `json:"followed_at"`     // 关注时间
}

// ProjectFollowerItem 项目关注者列表中的一条记录
type ProjectFollowerItem struct {
	UserBriefInfo
	FollowedAt time.Time `json:"followed_at"` // 关注时间
}

// FeedType 动态类型枚举
type FeedType string

const (
	FeedTypeLog      FeedType = "log"      // 被关注用户的开发日志
	FeedTypePost     FeedType = "post"     // 被关注用户的社区帖子
	FeedTypeTimeline FeedType = "timeline" // 被关注项目的时间轴事件
)

// FeedItem 动态流中的单条记录
type FeedItem struct {
	FeedType   FeedType    `json:"feed_type"`            // 动态类型：log 或 timeline
	CreatedAt  time.Time   `json:"created_at"`           // 时间，用于排序
	Data       interface{} `json:"data"`                 // 具体数据：*model.DevLog 或 *model.ProjectTimeline
}

// feedPostWithAuthor 动态流中的帖子数据，附带展开的作者字段（flat fields）
type feedPostWithAuthor struct {
	*model.Post
	AuthorNickname  string `json:"author_nickname"`
	AuthorUsername  string `json:"author_username"`
	AuthorAvatarURL string `json:"author_avatar_url,omitempty"`
}

// feedLogWithAuthor 动态流中的开发日志数据，附带展开的作者字段（flat fields）
type feedLogWithAuthor struct {
	*model.DevLog
	AuthorNickname  string `json:"author_nickname"`
	AuthorUsername  string `json:"author_username"`
	AuthorAvatarURL string `json:"author_avatar_url,omitempty"`
}

// ===========================
// FollowService 接口定义
// ===========================

// FollowService 关注与通知系统的服务接口
type FollowService interface {
	// ===== 依赖注入 =====

	// SetNotificationService 注入统一通知服务（由 router 在初始化时调用）。
	SetNotificationService(svc NotificationService)

	// ===== 辅助方法 =====

	// GetUserIDByUsername 根据用户名查询用户 ID（Handler 层辅助调用）
	GetUserIDByUsername(ctx context.Context, username string) (uint64, error)

	// ===== 用户关注 =====

	// FollowUser 关注用户（不能关注自己，用事务更新双方计数）
	FollowUser(ctx context.Context, followerID, followeeID uint64) error
	// UnfollowUser 取关用户（用事务更新双方计数）
	UnfollowUser(ctx context.Context, followerID, followeeID uint64) error
	// GetFollowers 获取分页粉丝列表（含用户基本信息）
	GetFollowers(ctx context.Context, userID uint64, page, pageSize int) ([]*FollowUserItem, int64, error)
	// GetFollowing 获取分页关注列表（含用户基本信息）
	GetFollowing(ctx context.Context, userID uint64, page, pageSize int) ([]*FollowUserItem, int64, error)
	// IsFollowingUser 判断 followerID 是否关注了 followeeID
	IsFollowingUser(ctx context.Context, followerID, followeeID uint64) (bool, error)

	// ===== 项目关注 =====

	// FollowProject 关注项目（更新 projects.follower_count +1）
	FollowProject(ctx context.Context, userID, projectID uint64) error
	// UnfollowProject 取关项目（更新 projects.follower_count -1）
	UnfollowProject(ctx context.Context, userID, projectID uint64) error
	// GetProjectFollowers 获取项目的分页关注者列表（含用户基本信息）
	GetProjectFollowers(ctx context.Context, projectID uint64, page, pageSize int) ([]*ProjectFollowerItem, int64, error)
	// IsFollowingProject 判断 userID 是否关注了 projectID
	IsFollowingProject(ctx context.Context, userID, projectID uint64) (bool, error)


	// ===== 动态流 =====

	// GetFeed 获取当前用户的分页动态流
	// 动态来源：关注用户的公开已发布日志 + 关注项目的时间轴事件
	// 在内存中归并排序后按时间倒序分页返回
	GetFeed(ctx context.Context, userID uint64, page, pageSize int) ([]*FeedItem, int64, error)
	// GetFeedByCursor 使用 cursor 游标分页获取动态流
	// cursor 为上一页末尾条目的 Unix 时间戳（秒），0 表示从最新开始
	// nextCursor=0 表示已到末尾
	GetFeedByCursor(ctx context.Context, userID uint64, cursor int64, pageSize int) ([]*FeedItem, int64, error)

	// ===== 话题关注操作 =====

	// FollowTopic 关注话题（幂等）
	FollowTopic(ctx context.Context, userID, topicID uint64) error
	// UnfollowTopic 取关话题
	UnfollowTopic(ctx context.Context, userID, topicID uint64) error
	// IsFollowingTopic 是否已关注话题
	IsFollowingTopic(ctx context.Context, userID, topicID uint64) (bool, error)
	// GetFollowedTopics 获取我关注的话题列表（分页）
	GetFollowedTopics(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Topic, int64, error)
}

// ===========================
// followService 实现
// ===========================

// followService 是 FollowService 接口的具体实现
type followService struct {
	followRepo  repository.FollowRepository
	userRepo    repository.UserRepository
	projectRepo repository.ProjectRepository
	teamRepo    repository.TeamRepository
	*NotificationClient
	devLogRepo  repository.DevLogRepository
	postRepo    repository.PostRepository
	db          *gorm.DB
}

// NewFollowService 创建 followService 实例
// db 参数用于执行关注计数的事务更新
func NewFollowService(
	followRepo repository.FollowRepository,
	userRepo repository.UserRepository,
	projectRepo repository.ProjectRepository,
	teamRepo repository.TeamRepository,
	devLogRepo repository.DevLogRepository,
	postRepo repository.PostRepository,
	db *gorm.DB,
) FollowService {
	svc := &followService{
		followRepo:  followRepo,
		userRepo:    userRepo,
		projectRepo: projectRepo,
		teamRepo:    teamRepo,
		devLogRepo:  devLogRepo,
		postRepo:    postRepo,
		db:          db,
		NotificationClient: &NotificationClient{},
	}
	return svc
}

// ==================== 辅助方法实现 ====================

// GetUserIDByUsername 根据用户名查询用户 ID（Handler 层辅助调用）
func (s *followService) GetUserIDByUsername(ctx context.Context, username string) (uint64, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

// ==================== 用户关注实现 ====================

// FollowUser 关注用户
// - 不能关注自己
// - 双方任意一方拉黑对方时，拒绝关注
// - 在单个事务中：插入关注记录 + 更新关注者的 following_count + 更新被关注者的 follower_count
func (s *followService) FollowUser(ctx context.Context, followerID, followeeID uint64) error {
	// 不能关注自己
	if followerID == followeeID {
		return apperrors.CodeError(apperrors.CodeFollowSelf)
	}

	// 确认被关注者存在
	if _, err := s.userRepo.GetUserByID(ctx, followeeID); err != nil {
		return err
	}

	// 在事务中执行：创建关注记录并更新双方计数
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建关注记录（通过临时 followRepository 操作事务 DB）
		txFollowRepo := repository.NewFollowRepository(tx)
		if err := txFollowRepo.FollowUser(ctx, followerID, followeeID); err != nil {
			return err
		}

		// 原子更新关注者的 following_count +1
		if err := tx.Model(&model.User{}).
			Where("id = ?", followerID).
			Update("following_count", gorm.Expr("following_count + 1")).Error; err != nil {
			return err
		}

		// 原子更新被关注者的 follower_count +1
		if err := tx.Model(&model.User{}).
			Where("id = ?", followeeID).
			Update("follower_count", gorm.Expr("follower_count + 1")).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	// 事务提交后：异步向被关注者发送通知（失败不影响主流程）
	go func() {
		follower, err := s.userRepo.GetUserByID(ctx, followerID)
		if err != nil {
			return
		}
		nickname := follower.Nickname
		if nickname == "" {
			nickname = follower.Username
		}
		s.Send(ctx, &SendNotificationReq{
			UserID:  followeeID,
			Type:    model.NotificationTypeNewFollower,
			Title:   "有新粉丝关注了你",
			Content: fmt.Sprintf("%s 关注了你", nickname),
			Metadata: map[string]interface{}{
				"follower_id":       followerID,
				"follower_username": follower.Username,
			},
		})
	}()

	metrics.UserActionTotal.WithLabelValues("follow").Inc()
	return nil
}

// UnfollowUser 取关用户
// - 在单个事务中：删除关注记录 + 更新关注者的 following_count（不低于0）+ 更新被关注者的 follower_count（不低于0）
func (s *followService) UnfollowUser(ctx context.Context, followerID, followeeID uint64) error {
	// 在事务中执行：删除关注记录并更新双方计数
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txFollowRepo := repository.NewFollowRepository(tx)
		if err := txFollowRepo.UnfollowUser(ctx, followerID, followeeID); err != nil {
			return err
		}

		// 原子更新关注者的 following_count -1（不低于0）
		if err := tx.Model(&model.User{}).
			Where("id = ? AND following_count > 0", followerID).
			Update("following_count", gorm.Expr("following_count - 1")).Error; err != nil {
			return err
		}

		// 原子更新被关注者的 follower_count -1（不低于0）
		if err := tx.Model(&model.User{}).
			Where("id = ? AND follower_count > 0", followeeID).
			Update("follower_count", gorm.Expr("follower_count - 1")).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetFollowers 获取分页粉丝列表（含用户基本信息）
func (s *followService) GetFollowers(ctx context.Context, userID uint64, page, pageSize int) ([]*FollowUserItem, int64, error) {
	offset := (page - 1) * pageSize
	records, total, err := s.followRepo.GetFollowers(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 批量查询用户信息，避免 N+1
	ids := make([]uint64, 0, len(records))
	for _, rec := range records {
		ids = append(ids, rec.FollowerID)
	}
	userMap, err := s.fetchUserMap(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*FollowUserItem, 0, len(records))
	for _, rec := range records {
		user, ok := userMap[rec.FollowerID]
		if !ok {
			continue // 用户可能已被删除
		}
		avatarURL := ""
		if user.AvatarKey != "" {
			avatarURL = presignURL(ctx, user.AvatarKey)
		}
		items = append(items, &FollowUserItem{
			UserBriefInfo: UserBriefInfo{
				ID:        user.ID,
				Username:  user.Username,
				Nickname:  user.Nickname,
				AvatarKey: user.AvatarKey,
				Bio:       user.Bio,
			},
			AvatarURL:      avatarURL,
			FollowersCount: user.FollowerCount,
			FollowingCount: user.FollowingCount,
			IsFollowing:    false,
			Projects:       []interface{}{},
			FollowedAt:     rec.CreatedAt,
		})
	}
	return items, total, nil
}

// GetFollowing 获取分页关注列表（含用户基本信息）
func (s *followService) GetFollowing(ctx context.Context, userID uint64, page, pageSize int) ([]*FollowUserItem, int64, error) {
	offset := (page - 1) * pageSize
	records, total, err := s.followRepo.GetFollowing(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 批量查询被关注者信息，避免 N+1
	ids := make([]uint64, 0, len(records))
	for _, rec := range records {
		ids = append(ids, rec.FollowedID)
	}
	userMap, err := s.fetchUserMap(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*FollowUserItem, 0, len(records))
	for _, rec := range records {
		user, ok := userMap[rec.FollowedID]
		if !ok {
			continue
		}
		avatarURL := ""
		if user.AvatarKey != "" {
			avatarURL = presignURL(ctx, user.AvatarKey)
		}
		items = append(items, &FollowUserItem{
			UserBriefInfo: UserBriefInfo{
				ID:        user.ID,
				Username:  user.Username,
				Nickname:  user.Nickname,
				AvatarKey: user.AvatarKey,
				Bio:       user.Bio,
			},
			AvatarURL:      avatarURL,
			FollowersCount: user.FollowerCount,
			FollowingCount: user.FollowingCount,
			IsFollowing:    false,
			Projects:       []interface{}{},
			FollowedAt:     rec.CreatedAt,
		})
	}
	return items, total, nil
}

// IsFollowingUser 判断 followerID 是否关注了 followeeID
func (s *followService) IsFollowingUser(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	return s.followRepo.IsFollowingUser(ctx, followerID, followeeID)
}

// ==================== 项目关注实现 ====================

// FollowProject 关注项目，更新 projects.follower_count +1
func (s *followService) FollowProject(ctx context.Context, userID, projectID uint64) error {
	// 确认项目存在
	if _, err := s.projectRepo.GetProjectByID(ctx, projectID); err != nil {
		return err
	}
	// 创建关注记录（幼等）
	if err := s.followRepo.FollowProject(ctx, userID, projectID); err != nil {
		return err
	}
	// 项目关注数 +1（原子操作，容忍失败不回滚）
	if err := s.projectRepo.IncrFollowerCount(ctx, projectID, 1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// UnfollowProject 取关项目，更新 projects.follower_count -1
func (s *followService) UnfollowProject(ctx context.Context, userID, projectID uint64) error {
	// 删除关注记录
	if err := s.followRepo.UnfollowProject(ctx, userID, projectID); err != nil {
		return err
	}
	// 项目关注数 -1（原子操作，容忍失败不回滚）
	if err := s.projectRepo.IncrFollowerCount(ctx, projectID, -1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// GetProjectFollowers 获取项目的分页关注者列表（含用户基本信息）
func (s *followService) GetProjectFollowers(ctx context.Context, projectID uint64, page, pageSize int) ([]*ProjectFollowerItem, int64, error) {
	offset := (page - 1) * pageSize
	records, total, err := s.followRepo.GetProjectFollowers(ctx, projectID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 批量查询关注者信息，避免 N+1
	ids := make([]uint64, 0, len(records))
	for _, rec := range records {
		ids = append(ids, rec.UserID)
	}
	userMap, err := s.fetchUserMap(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*ProjectFollowerItem, 0, len(records))
	for _, rec := range records {
		user, ok := userMap[rec.UserID]
		if !ok {
			continue
		}
		items = append(items, &ProjectFollowerItem{
			UserBriefInfo: UserBriefInfo{
				ID:        user.ID,
				Username:  user.Username,
				Nickname:  user.Nickname,
				AvatarKey: user.AvatarKey,
				Bio:       user.Bio,
			},
			FollowedAt: rec.CreatedAt,
		})
	}
	return items, total, nil
}

// fetchUserMap 批量获取用户信息并返回 id→user 映射（follow_service 内部辅助函数）
func (s *followService) fetchUserMap(ctx context.Context, ids []uint64) (map[uint64]*model.User, error) {
	if len(ids) == 0 {
		return map[uint64]*model.User{}, nil
	}
	users, err := s.userRepo.GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	m := make(map[uint64]*model.User, len(users))
	for _, u := range users {
		m[u.ID] = u
	}
	return m, nil
}

// IsFollowingProject 判断 userID 是否关注了 projectID
func (s *followService) IsFollowingProject(ctx context.Context, userID, projectID uint64) (bool, error) {
	return s.followRepo.IsFollowingProject(ctx, userID, projectID)
}

// ==================== 动态流实现 ====================

// feedMergeItem 内部归并排序用的中间结构
type feedMergeItem struct {
	createdAt time.Time
	feedType  FeedType
	data      interface{}
}

// GetFeed 获取当前用户的动态流（分页）。
//
// 推拉结合策略：
//  1. 优先尝试从 Redis ZSet（写扩散缓存）读取足够数量的条目，快速返回
//  2. 查询关注列表中的大V，拉取其最新内容进行读合并（推拉结合）
//  3. 若缓存不足（新用户、冷启动、ZSet 已过期），fallback 到原有全量拉取逻辑
//
// 写扩散缓存由 feedFanout 在内容发布时异步维护。
func (s *followService) GetFeed(ctx context.Context, userID uint64, page, pageSize int) ([]*FeedItem, int64, error) {
	offset := (page - 1) * pageSize

	// 1. 从 Redis ZSet 读取普通用户写扩散的内容
	cachedMembers, cacheErr := feedGetFromCache(ctx, userID, offset, pageSize*3)

	// 2. 查询关注列表中的大V，拉取其内容（推拉结合：读合并部分）
	var heavyEntries []feedMergeEntry
	_, heavyIDs, partErr := s.followRepo.GetFollowingIDsPartitioned(ctx, userID, cache.FanoutThreshold)
	if partErr == nil && len(heavyIDs) > 0 {
		heavyEntries = feedFetchHeavyUserLogs(ctx, s.devLogRepo, heavyIDs, pageSize*2)
		heavyEntries = append(heavyEntries, feedFetchHeavyUserPosts(ctx, s.postRepo, heavyIDs, pageSize*2)...)
		metrics.FeedMergeTotal.WithLabelValues("heavy_merged").Inc()
	}

	// 3. 有缓存命中或有大V内容时，构建混合结果
	if cacheErr == nil && (len(cachedMembers) > 0 || len(heavyEntries) > 0) {
		items, err := s.buildFeedMixed(ctx, cachedMembers, heavyEntries, pageSize)
		if err == nil && len(items) > 0 {
			total, _ := feedCountInCache(ctx, userID)
			return items, total + int64(len(heavyEntries)), nil
		}
	}

	// 4. 完全 fallback（新用户/冷启动/缓存完全过期）
	metrics.FeedMergeTotal.WithLabelValues("cache_miss").Inc()
	return s.getFeedLegacy(ctx, userID, page, pageSize)
}

// getFeedLegacy 原有的全量拉取 + 内存归并逻辑（作为 fallback）。
// 当 Redis ZSet 缓存不足时使用，完整保留原始实现。
func (s *followService) getFeedLegacy(ctx context.Context, userID uint64, page, pageSize int) ([]*FeedItem, int64, error) {
	// 1. 获取关注的用户 ID 和项目 ID
	followingUserIDs, err := s.followRepo.GetFollowingIDs(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	followingProjectIDs, err := s.followRepo.GetFollowingProjectIDs(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	// 如果没有任何关注，直接返回空列表
	if len(followingUserIDs) == 0 && len(followingProjectIDs) == 0 {
		return []*FeedItem{}, 0, nil
	}

	// 2. 批量拉取动态数据
	// 为了支持分页排序，拉取足够多条数据后在内存中排序
	// 每类最多拉取 feedMaxFetchPerType 条，覆盖常规分页场景
	const feedMaxFetchPerType = 500

	allItems := make([]feedMergeItem, 0, feedMaxFetchPerType*2)

	// 2a. 拉取被关注用户的公开已发布开发日志
	if len(followingUserIDs) > 0 {
		logs, err := s.devLogRepo.ListPublicLogsByAuthorIDs(ctx, followingUserIDs, feedMaxFetchPerType)
		if err != nil {
			return nil, 0, err
		}
		for _, log := range logs {
			allItems = append(allItems, feedMergeItem{
				createdAt: log.CreatedAt,
				feedType:  FeedTypeLog,
				data:      log,
			})
		}
	}

	// 2a.5. 拉取被关注用户的帖子
	if len(followingUserIDs) > 0 {
		posts, err := s.postRepo.ListPublicPostsByAuthorIDs(ctx, followingUserIDs, feedMaxFetchPerType)
		if err != nil {
			return nil, 0, err
		}
		for _, p := range posts {
			allItems = append(allItems, feedMergeItem{
				createdAt: p.CreatedAt,
				feedType:  FeedTypePost,
				data:      p,
			})
		}
	}

	// 2b. 拉取被关注项目的时间轴事件
	if len(followingProjectIDs) > 0 {
		timelines, err := s.projectRepo.ListTimelinesByProjectIDs(ctx, followingProjectIDs, feedMaxFetchPerType)
		if err != nil {
			return nil, 0, err
		}
		for _, t := range timelines {
			allItems = append(allItems, feedMergeItem{
				createdAt: t.CreatedAt,
				feedType:  FeedTypeTimeline,
				data:      t,
			})
		}
	}

	// 3. 在内存中按时间倒序排序
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].createdAt.After(allItems[j].createdAt)
	})

	// 4. 计算总数并按页码分页
	total := int64(len(allItems))
	offset := (page - 1) * pageSize
	if offset >= len(allItems) {
		return []*FeedItem{}, total, nil
	}
	end := offset + pageSize
	if end > len(allItems) {
		end = len(allItems)
	}
	pageItems := allItems[offset:end]

	// 5. 构造结果
	result := make([]*FeedItem, 0, len(pageItems))
	for _, item := range pageItems {
		result = append(result, &FeedItem{
			FeedType:  item.feedType,
			CreatedAt: item.createdAt,
			Data:      item.data,
		})
	}

	s.enrichFeedAuthors(ctx, result)
	return result, total, nil
}

// buildFeedFromCache 根据 Redis ZSet 中的 member 字符串列表批量构建 FeedItem 切片。
// member 格式为 "{type}:{id}"，支持 "log" 和 "timeline" 两种类型。
// 使用批量查询，避免逐条 GetLogByID/GetTimelineByID 产生的 N+1。
func (s *followService) buildFeedFromCache(ctx context.Context, members []string) ([]*FeedItem, error) {
	if len(members) == 0 {
		return []*FeedItem{}, nil
	}

	// 分类收集 ID
	logIDs := make([]uint64, 0)
	timelineIDs := make([]uint64, 0)
	postIDs := make([]uint64, 0)
	memberOrder := make([]struct{ typ string; id uint64 }, 0, len(members))

	for _, member := range members {
		contentType, contentID, err := parseFeedMember(member)
		if err != nil {
			continue
		}
		memberOrder = append(memberOrder, struct{ typ string; id uint64 }{contentType, contentID})
		switch contentType {
		case "log":
			logIDs = append(logIDs, contentID)
		case "timeline":
			timelineIDs = append(timelineIDs, contentID)
		case "post":
			postIDs = append(postIDs, contentID)
		}
	}

	// 批量查询
	logMap := make(map[uint64]*model.DevLog)
	if len(logIDs) > 0 {
		logs, err := s.devLogRepo.GetLogsByIDs(ctx, logIDs)
		if err == nil {
			for _, l := range logs {
				logMap[l.ID] = l
			}
		}
	}
	timelineMap := make(map[uint64]*model.ProjectTimeline)
	if len(timelineIDs) > 0 {
		events, err := s.projectRepo.GetTimelinesByIDs(ctx, timelineIDs)
		if err == nil {
			for _, e := range events {
				timelineMap[e.ID] = e
			}
		}
	}

	// 按原始顺序组装结果
	postMap := make(map[uint64]*model.Post)
	if len(postIDs) > 0 {
		posts, err := s.postRepo.GetPostsByIDs(ctx, postIDs)
		if err == nil {
			for _, p := range posts {
				postMap[p.ID] = p
			}
		}
	}

	result := make([]*FeedItem, 0, len(memberOrder))
	for _, mo := range memberOrder {
		switch mo.typ {
		case "log":
			devLog, ok := logMap[mo.id]
			if !ok {
				continue // 已删除
			}
			if devLog.Status != "published" || devLog.Visibility != "public" {
				continue // 二次过滤
			}
			result = append(result, &FeedItem{
				FeedType:  FeedTypeLog,
				CreatedAt: devLog.CreatedAt,
				Data:      devLog,
			})
		case "timeline":
			event, ok := timelineMap[mo.id]
			if !ok {
				continue
			}
			result = append(result, &FeedItem{
				FeedType:  FeedTypeTimeline,
				CreatedAt: event.CreatedAt,
				Data:      event,
			})
		case "post":
			p, ok := postMap[mo.id]
			if !ok {
				continue
			}
			if p.DeletedAt != nil {
				continue
			}
			result = append(result, &FeedItem{
				FeedType:  FeedTypePost,
				CreatedAt: p.CreatedAt,
				Data:      p,
			})
		}
	}
	s.enrichFeedAuthors(ctx, result)
	return result, nil
}

// buildFeedMixed 合并 ZSet 缓存条目（普通用户写扩散）和大V日志条目，按时间倒序返回。
func (s *followService) buildFeedMixed(ctx context.Context, cachedMembers []string, heavyEntries []feedMergeEntry, limit int) ([]*FeedItem, error) {
	cacheItems, _ := s.buildFeedFromCache(ctx, cachedMembers)

	for _, entry := range heavyEntries {
		switch entry.contentType {
		case "log":
			devLog, err := s.devLogRepo.GetLogByID(ctx, entry.contentID)
			if err != nil || devLog == nil {
				continue
			}
			if string(devLog.Status) != "published" || string(devLog.Visibility) != "public" {
				continue
			}
			cacheItems = append(cacheItems, &FeedItem{
				FeedType:  FeedTypeLog,
				CreatedAt: devLog.CreatedAt,
				Data:      devLog,
			})
		case "post":
			p, err := s.postRepo.GetPostByID(ctx, entry.contentID)
			if err != nil || p == nil || p.DeletedAt != nil {
				continue
			}
			cacheItems = append(cacheItems, &FeedItem{
				FeedType:  FeedTypePost,
				CreatedAt: p.CreatedAt,
				Data:      p,
			})
		}
	}

	sort.Slice(cacheItems, func(i, j int) bool {
		return cacheItems[i].CreatedAt.After(cacheItems[j].CreatedAt)
	})
	if len(cacheItems) > limit {
		cacheItems = cacheItems[:limit]
	}
	s.enrichFeedAuthors(ctx, cacheItems)
	return cacheItems, nil
}

// GetFeedByCursor 使用 score 游标从 Redis ZSet 分页读取动态流。
// cursor = 上一页最后一条的 Unix 时间戳（score），0 表示从最新开始。
// 若 ZSet 不足则 fallback 到 getFeedLegacy（取第一页）。
func (s *followService) GetFeedByCursor(ctx context.Context, userID uint64, cursor int64, pageSize int) ([]*FeedItem, int64, error) {
	key := cache.UserFeedKey(userID)
	rdb := cache.Get()
	if rdb == nil {
		return nil, 0, fmt.Errorf("redis not available")
	}

	// 用 ZREVRANGEBYSCORE：score < cursor（若 cursor=0 则 +inf）
	var maxScore string
	if cursor == 0 {
		maxScore = "+inf"
	} else {
		maxScore = fmt.Sprintf("(%d", cursor) // 不含 cursor 本身
	}

	members, err := rdb.ZRevRangeByScoreWithScores(ctx, key, &goredis.ZRangeBy{
		Max:    maxScore,
		Min:    "-inf",
		Offset: 0,
		Count:  int64(pageSize + 1), // 多取 1 条判断是否有下一页
	}).Result()

	// Build cache items from Redis ZSet (push-based fanout)
	var cacheItems []*FeedItem
	if err == nil && len(members) > 0 {
		if len(members) > pageSize {
			members = members[:pageSize]
		}

		memberStrs := make([]string, len(members))
		for i, m := range members {
			memberStrs[i] = m.Member.(string)
		}
		cacheItems, err = s.buildFeedFromCache(ctx, memberStrs)
		if err != nil {
			return nil, 0, err
		}
	}

	// ALWAYS pull recent content from DB for items not yet fanned out
	legacyItems, _, legacyErr := s.getFeedLegacy(ctx, userID, 1, pageSize*2)
	if legacyErr != nil {
		return nil, 0, legacyErr
	}

	// Merge cacheItems + legacyItems, deduplicate by composite key {FeedType}:{ID}
	seen := make(map[string]bool)
	merged := make([]*FeedItem, 0, len(cacheItems)+len(legacyItems))

	for _, item := range legacyItems {
		key := fmt.Sprintf("%s:%d", item.FeedType, feedItemID(item.Data))
		if !seen[key] {
			seen[key] = true
			merged = append(merged, item)
		}
	}
	for _, item := range cacheItems {
		key := fmt.Sprintf("%s:%d", item.FeedType, feedItemID(item.Data))
		if !seen[key] {
			seen[key] = true
			merged = append(merged, item)
		}
	}

	// Sort by CreatedAt descending
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].CreatedAt.After(merged[j].CreatedAt)
	})

	// Slice to pageSize
	if len(merged) > pageSize {
		merged = merged[:pageSize]
	}

	return merged, 0, nil
}

// feedItemID extracts the numeric ID from a FeedItem.Data value.
func feedItemID(data interface{}) uint64 {
	switch v := data.(type) {
	case *model.DevLog:
		return v.ID
	case *feedLogWithAuthor:
		return v.ID
	case *model.Post:
		return v.ID
	case *feedPostWithAuthor:
		return v.ID
	case *model.ProjectTimeline:
		return v.ID
	default:
		return 0
	}
}

// enrichFeedAuthors 批量查询 FeedItem 中日志和帖子的作者信息，并将原始 Data
// 替换为附带 flat author 字段的包装结构（feedLogWithAuthor / feedPostWithAuthor）。
// 已包装过的条目（类型为 feedLogWithAuthor / feedPostWithAuthor）会被跳过。
func (s *followService) enrichFeedAuthors(ctx context.Context, items []*FeedItem) {
	// 1. 收集所有需要查询的作者 ID
	authorIDs := make(map[uint64]bool)
	for _, item := range items {
		switch item.FeedType {
		case FeedTypeLog:
			if l, ok := item.Data.(*model.DevLog); ok {
				authorIDs[l.AuthorID] = true
			}
		case FeedTypePost:
			if p, ok := item.Data.(*model.Post); ok {
				authorIDs[p.AuthorID] = true
			}
		}
	}
	if len(authorIDs) == 0 {
		return
	}

	// 2. 批量查询用户
	ids := make([]uint64, 0, len(authorIDs))
	for id := range authorIDs {
		ids = append(ids, id)
	}
	userMap, err := s.fetchUserMap(ctx, ids)
	if err != nil {
		logger.Warn("failed to batch fetch authors for feed", zap.Error(err))
		return
	}

	// 3. 替换 Data 为带作者信息的包装结构
	for i, item := range items {
		switch item.FeedType {
		case FeedTypeLog:
			if l, ok := item.Data.(*model.DevLog); ok {
				if u, ok2 := userMap[l.AuthorID]; ok2 {
					avatarURL := ""
					if u.AvatarKey != "" {
						avatarURL = presignURL(ctx, u.AvatarKey)
					}
					items[i].Data = &feedLogWithAuthor{
						DevLog:          l,
						AuthorNickname:  u.Nickname,
						AuthorUsername:  u.Username,
						AuthorAvatarURL: avatarURL,
					}
				}
			}
		case FeedTypePost:
			if p, ok := item.Data.(*model.Post); ok {
				if u, ok2 := userMap[p.AuthorID]; ok2 {
					avatarURL := ""
					if u.AvatarKey != "" {
						avatarURL = presignURL(ctx, u.AvatarKey)
					}
					items[i].Data = &feedPostWithAuthor{
						Post:            p,
						AuthorNickname:  u.Nickname,
						AuthorUsername:  u.Username,
						AuthorAvatarURL: avatarURL,
					}
				}
			}
		}
	}
}

// ===========================
// Topic 关注实现
// ===========================

// FollowTopic 关注话题：写入 topic_follows，更新冗余字段
func (s *followService) FollowTopic(ctx context.Context, userID, topicID uint64) error {
	// 检查是否已关注（幂等保护）
	already, err := s.followRepo.IsFollowingTopic(ctx, userID, topicID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if already {
		return nil // 已关注，幂等
	}
	if err := s.followRepo.FollowTopic(ctx, userID, topicID); err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	// 冗余字段 +1（容忍失败，不影响主流程）
	if err := s.followRepo.IncrTopicFollowerCount(ctx, topicID, 1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// UnfollowTopic 取关话题
func (s *followService) UnfollowTopic(ctx context.Context, userID, topicID uint64) error {
	following, err := s.followRepo.IsFollowingTopic(ctx, userID, topicID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if !following {
		return nil // 未关注，幂等
	}
	if err := s.followRepo.UnfollowTopic(ctx, userID, topicID); err != nil {
		return apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	if err := s.followRepo.IncrTopicFollowerCount(ctx, topicID, -1); err != nil {
		logger.Warn("failed to update counts", zap.Error(err))
	}
	return nil
}

// IsFollowingTopic 是否已关注话题
func (s *followService) IsFollowingTopic(ctx context.Context, userID, topicID uint64) (bool, error) {
	return s.followRepo.IsFollowingTopic(ctx, userID, topicID)
}

// GetFollowedTopics 获取我关注的话题列表（分页）
func (s *followService) GetFollowedTopics(ctx context.Context, userID uint64, page, pageSize int) ([]*model.Topic, int64, error) {
	offset := (page - 1) * pageSize
	topics, total, err := s.followRepo.GetFollowedTopics(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.CodeInternalError, err)
	}
	return topics, total, nil
}


// ClearNotifications 清空当前用户所有通知
func (s *followService) ClearNotifications(ctx context.Context, userID uint64) error {
	if err := s.teamRepo.ClearNotifications(ctx, userID); err != nil {
		return err
	}
	cacheKey := cache.UserUnreadKey(userID)
	if err := cache.Del(ctx, cacheKey); err != nil {
		logger.Warn("failed to delete cache", zap.Error(err))
	}
	return nil
}
