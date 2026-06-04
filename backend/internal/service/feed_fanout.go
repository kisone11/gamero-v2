// Package service — Feed 流写扩散辅助函数
// 提供推拉结合架构中的写扩散（Fanout-on-write）核心逻辑：
//   - feedFanout: 批量把一条内容推入粉丝的 Feed ZSet
//   - feedGetFromCache / feedCountInCache: 从 Redis 读取 Feed
//   - parseFeedMember: 解析 ZSet member 格式 "{type}:{id}"
//   - feedFetchHeavyUserLogs: 大V读合并（推拉结合的"拉"部分）
package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// feedFanout 写扩散：将内容推送到指定粉丝列表的 Feed ZSet。
//
// 参数说明：
//   - contentType: "log" | "timeline"
//   - contentID:   内容唯一 ID
//   - score:       写入 ZSet 的分数，通常为内容发布时间的 Unix 秒时间戳
//   - followerIDs: 需要推送的粉丝 ID 列表（调用方负责只传粉丝数 < FanoutThreshold 的用户）
//
// 写扩散失败不影响主流程——通过 goroutine 异步调用，只记录日志。
func feedFanout(ctx context.Context, contentType string, contentID uint64, score float64, followerIDs []uint64) {
	if len(followerIDs) == 0 {
		return
	}

	member := fmt.Sprintf("%s:%d", contentType, contentID)
	rdb := cache.Get()
	if rdb == nil {
		return
	}
	pipe := rdb.Pipeline()

	for _, followerID := range followerIDs {
		key := cache.UserFeedKey(followerID)
		// 写入成员（重复写入幂等，ZSet 自动更新 score）
		pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
		// 保持 ZSet 最多 FeedMaxLen 条，裁剪最老的（rank 0 到 -(FeedMaxLen+1)）
		pipe.ZRemRangeByRank(ctx, key, 0, -(cache.FeedMaxLen + 1))
	}

	if _, err := pipe.Exec(ctx); err != nil {
		// 写扩散失败不影响主流程，只记录日志
		logger.Warn("feed fanout pipeline failed",
			zap.String("contentType", contentType),
			zap.Uint64("contentID", contentID),
			zap.Int("followerCount", len(followerIDs)),
			zap.Error(err),
		)
	}
}

// feedGetFromCache 从用户的 Feed ZSet 按倒序（最新在前）读取成员字符串列表。
//
// 返回的每个字符串格式为 "{type}:{id}"，例如 "log:123"、"timeline:456"。
// offset / limit 对应分页的起始索引和数量。
func feedGetFromCache(ctx context.Context, userID uint64, offset, limit int) ([]string, error) {
	rdb := cache.Get()
	if rdb == nil {
		return nil, nil
	}
	key := cache.UserFeedKey(userID)
	start := int64(offset)
	stop := int64(offset + limit - 1)
	return rdb.ZRevRange(ctx, key, start, stop).Result()
}

// feedCountInCache 返回用户 Feed ZSet 的总成员数（用于估算分页 total）。
func feedCountInCache(ctx context.Context, userID uint64) (int64, error) {
	rdb := cache.Get()
	if rdb == nil {
		return 0, nil
	}
	return rdb.ZCard(ctx, cache.UserFeedKey(userID)).Result()
}

// parseFeedMember 将 ZSet member 字符串 "{type}:{id}" 拆解为类型和 ID。
//
// 支持格式：
//   - "log:123"      → contentType="log", contentID=123
//   - "timeline:456" → contentType="timeline", contentID=456
func parseFeedMember(member string) (contentType string, contentID uint64, err error) {
	for i, c := range member {
		if c == ':' {
			contentType = member[:i]
			id, parseErr := strconv.ParseUint(member[i+1:], 10, 64)
			if parseErr != nil {
				return "", 0, fmt.Errorf("parseFeedMember: invalid id in %q: %w", member, parseErr)
			}
			return contentType, id, nil
		}
	}
	return "", 0, fmt.Errorf("parseFeedMember: missing ':' in %q", member)
}

// feedMergeEntry 大V内容合并条目（用于推拉结合的读扩散部分）
type feedMergeEntry struct {
	contentType string
	contentID   uint64
	score       float64
}

// feedFetchHeavyUserLogs 拉取大V用户的最新公开日志，用于读合并。
// 这是推拉结合架构的"拉"部分：大V不做写扩散，读取 Feed 时合并。
func feedFetchHeavyUserLogs(ctx context.Context, devLogRepo repository.DevLogRepository, heavyAuthorIDs []uint64, limit int) []feedMergeEntry {
	if len(heavyAuthorIDs) == 0 {
		return nil
	}
	logs, err := devLogRepo.ListPublicLogsByAuthorIDs(ctx, heavyAuthorIDs, limit)
	if err != nil {
		return nil
	}
	entries := make([]feedMergeEntry, 0, len(logs))
	for _, log := range logs {
		entries = append(entries, feedMergeEntry{
			contentType: "log",
			contentID:   log.ID,
			score:       float64(log.CreatedAt.Unix()),
		})
	}
	return entries
}

// feedFetchHeavyUserPosts 拉取大V用户的最新公开帖子，用于读合并。
// 这是推拉结合架构的"拉"部分：大V不做写扩散，读取 Feed 时合并。
func feedFetchHeavyUserPosts(ctx context.Context, postRepo repository.PostRepository, heavyAuthorIDs []uint64, limit int) []feedMergeEntry {
	if len(heavyAuthorIDs) == 0 {
		return nil
	}
	posts, err := postRepo.ListPublicPostsByAuthorIDs(ctx, heavyAuthorIDs, limit)
	if err != nil {
		return nil
	}
	entries := make([]feedMergeEntry, 0, len(posts))
	for _, p := range posts {
		entries = append(entries, feedMergeEntry{
			contentType: "post",
			contentID:   p.ID,
			score:       float64(p.CreatedAt.Unix()),
		})
	}
	return entries
}
