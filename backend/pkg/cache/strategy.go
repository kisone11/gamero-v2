// Package cache — strategy.go 提供缓存 key 生成规范、TTL 策略常量、
// 批量删除 pattern keys 工具方法和缓存预热接口。
// 与 cache.go（基础 Redis 操作）和 keys.go（已有 key 常量）协同使用。
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
)

// ===========================
// Key 生成规范
// ===========================
// 统一格式：prefix:entity:id:field
// 例：gamero:user:123:profile
//     gamero:project:42:stats
//     gamero:notif:99:silence

const (
	// KeyPrefixGlobal 所有 key 的统一前缀（方便多应用共享同一 Redis 时隔离命名空间）
	KeyPrefixGlobal = "gamero"

	// ─── 实体前缀 ───
	keyEntityUser        = "user"
	keyEntityProject     = "project"
	keyEntityPost        = "post"
	keyEntityNotif       = "notif"
	keyEntityTeam        = "team"
	keyEntityRecruitment = "recruitment"
)

// ─── 追加 TTL 常量（补充 keys.go 中已有的 TTL 定义）───

const (
	// TTLProjectStats 项目统计缓存（成员变动/发布 release 时主动失效）
	TTLProjectStats = 15 * time.Minute
	// TTLUserSearchWeight 用户搜索权重缓存（活跃度得分，1小时更新一次）
	TTLUserSearchWeight = 1 * time.Hour
	// TTLNotifSilence 通知静默配置缓存（用户可设，10分钟同步一次）
	TTLNotifSilence = 10 * time.Minute
	// TTLTeamRecruitmentExpiry 招募有效期检查的标记 key
	TTLTeamRecruitmentExpiry = 24 * time.Hour
	// TTLSlugGenLock slug 生成分布式锁超时时间
	TTLSlugGenLock = 5 * time.Second
	// TTLBatchOpLock 批量操作防重锁
	TTLBatchOpLock = 30 * time.Second
)

// ===========================
// 结构化 Key 生成函数
// ===========================

// BuildKey 按 prefix:entity:id:field 规范生成缓存 key。
// field 可为空字符串，此时 key 格式为 prefix:entity:id。
//
//	BuildKey("user", "123", "profile")   → "gamero:user:123:profile"
//	BuildKey("project", "42", "stats")   → "gamero:project:42:stats"
//	BuildKey("notif", "99", "")          → "gamero:notif:99"
func BuildKey(entity, id, field string) string {
	if field == "" {
		return fmt.Sprintf("%s:%s:%s", KeyPrefixGlobal, entity, id)
	}
	return fmt.Sprintf("%s:%s:%s:%s", KeyPrefixGlobal, entity, id, field)
}

// ProjectStatsKey 项目统计信息缓存 key
func ProjectStatsKey(projectID uint64) string {
	return BuildKey(keyEntityProject, fmt.Sprintf("%d", projectID), "stats")
}

// ProjectMemberCountKey 项目成员数缓存 key（成员变动时失效）
func ProjectMemberCountKey(projectID uint64) string {
	return BuildKey(keyEntityProject, fmt.Sprintf("%d", projectID), "member_count")
}

// ProjectReleaseCountKey 项目版本发布数缓存 key（发布 release 时失效）
func ProjectReleaseCountKey(projectID uint64) string {
	return BuildKey(keyEntityProject, fmt.Sprintf("%d", projectID), "release_count")
}

// NotifSilenceKey 用户通知静默配置缓存 key
func NotifSilenceKey(userID uint64) string {
	return BuildKey(keyEntityNotif, fmt.Sprintf("%d", userID), "silence")
}

// UserSearchWeightKey 用户搜索权重缓存 key
func UserSearchWeightKey(userID uint64) string {
	return BuildKey(keyEntityUser, fmt.Sprintf("%d", userID), "search_weight")
}

// RecruitmentExpiredFlagKey 招募已过期标记 key（避免重复处理）
func RecruitmentExpiredFlagKey(recruitmentID uint64) string {
	return BuildKey(keyEntityRecruitment, fmt.Sprintf("%d", recruitmentID), "expired_flag")
}

// SlugGenLockKey slug 生成分布式锁 key
func SlugGenLockKey(slug string) string {
	return fmt.Sprintf("%s:lock:slug:%s", KeyPrefixGlobal, slug)
}

// BatchOpLockKey 批量操作防重锁 key
func BatchOpLockKey(opName string, targetID uint64) string {
	return fmt.Sprintf("%s:lock:batch:%s:%d", KeyPrefixGlobal, opName, targetID)
}

// ===========================
// 批量删除 Pattern Keys
// ===========================

// DeleteByPattern 使用 SCAN 游标安全地删除匹配 pattern 的所有 key。
// 使用 SCAN 替代 KEYS，避免生产环境阻塞。
// 若没有匹配的 key，静默返回 nil。
//
// 示例：
//
//	DeleteByPattern(ctx, "gamero:project:42:*")
//	DeleteByPattern(ctx, "gamero:user:*:search_weight")
func DeleteByPattern(ctx context.Context, pattern string) error {
	if Get() == nil {
		return nil
	}

	var cursor uint64
	var deleted int64

	for {
		keys, nextCursor, err := Scan(ctx, cursor, pattern, 100)
		if err != nil {
			return fmt.Errorf("cache.DeleteByPattern scan failed: %w", err)
		}

		if len(keys) > 0 {
			if err := Del(ctx, keys...); err != nil {
				logger.Warn("cache.DeleteByPattern: failed to delete keys batch",
					zap.String("pattern", pattern),
					zap.Int("count", len(keys)),
					zap.Error(err),
				)
			} else {
				deleted += int64(len(keys))
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if deleted > 0 {
		logger.Info("cache.DeleteByPattern: keys deleted",
			zap.String("pattern", pattern),
			zap.Int64("count", deleted),
		)
	}

	return nil
}

// InvalidateProjectCache 失效项目相关的所有缓存（成员变动/发布 release 时调用）。
// 批量删除：项目详情、统计、成员数、版本数等。
func InvalidateProjectCache(ctx context.Context, projectID uint64) {
	keys := []string{
		ProjectStatsKey(projectID),
		ProjectMemberCountKey(projectID),
		ProjectReleaseCountKey(projectID),
		// 项目详情（slug-based）此处用 projectID 路径，handler 层知道 slug 时应额外删除
		BuildKey(keyEntityProject, fmt.Sprintf("%d", projectID), "detail"),
	}
	if err := Del(ctx, keys...); err != nil {
		logger.Warn("cache.InvalidateProjectCache: failed",
			zap.Uint64("project_id", projectID),
			zap.Error(err),
		)
	}
}

// InvalidateUserCache 失效用户相关缓存（资料变动/关注数变化时调用）。
func InvalidateUserCache(ctx context.Context, userID uint64) {
	keys := []string{
		UserProfileKey(userID),
		UserSearchWeightKey(userID),
	}
	if err := Del(ctx, keys...); err != nil {
		logger.Warn("cache.InvalidateUserCache: failed",
			zap.Uint64("user_id", userID),
			zap.Error(err),
		)
	}
}

// ===========================
// 缓存预热接口
// ===========================

// Warmer 缓存预热接口。
// 实现者负责将"冷启动"时频繁被访问的数据预加载到 Redis，
// 以避免缓存击穿（大量请求同时打到 DB）。
type Warmer interface {
	// Warm 执行预热逻辑。ctx 支持超时取消。
	// 应在应用启动完成、DB/Redis 就绪后调用。
	Warm(ctx context.Context) error

	// Name 返回预热器的可识别名称（用于日志）
	Name() string
}

// RunWarmers 依次执行所有预热器。任意预热器失败不阻断后续预热。
// 预热失败只记录 Warn 日志，不影响应用启动（降级安全）。
func RunWarmers(ctx context.Context, warmers []Warmer) {
	for _, w := range warmers {
		name := w.Name()
		logger.Info("cache warm: starting", zap.String("warmer", name))
		start := time.Now()
		if err := w.Warm(ctx); err != nil {
			logger.Warn("cache warm: failed",
				zap.String("warmer", name),
				zap.Duration("elapsed", time.Since(start)),
				zap.Error(err),
			)
		} else {
			logger.Info("cache warm: done",
				zap.String("warmer", name),
				zap.Duration("elapsed", time.Since(start)),
			)
		}
	}
}

// NoopWarmer 空实现预热器（用于测试或占位）
type NoopWarmer struct {
	name string
}

// NewNoopWarmer 创建一个空实现预热器
func NewNoopWarmer(name string) *NoopWarmer {
	return &NoopWarmer{name: name}
}

// Warm 空预热，直接返回 nil
func (w *NoopWarmer) Warm(_ context.Context) error { return nil }

// Name 返回预热器名称
func (w *NoopWarmer) Name() string { return w.name }
