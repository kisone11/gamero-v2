package cache

import (
	"fmt"
	"time"
)

// ===========================
// Redis Key 前缀规范
// ===========================

const (
	KeyUserProfile    = "user:profile:%d"    // %d = userID
	KeyUserUnreadCnt  = "user:unread:%d"     // %d = userID
	KeyUserNotifCats  = "user:notif:cats:%d" // %d = userID 通知分类未读数
	KeyPostDetail     = "post:detail:%d"     // %d = postID
	KeyProjectDetail  = "project:detail:%s"  // %s = slug
	KeyFollowRelation = "follow:rel:%d:%d"   // follower:followee
	KeyLoginFailCnt   = "login:fail:%s"      // %s = email/phone，暴力破解防护
)

// ===========================
// 统一 TTL 常量（time.Duration）
// ===========================

const (
	TTLUserProfile     = 5 * time.Minute  // 用户主页
	TTLPostDetail      = 3 * time.Minute  // 帖子详情
	TTLHotList         = 1 * time.Hour    // 热门列表
	TTLProjectDetail   = 10 * time.Minute // 项目详情
	TTLFollowRelation  = 30 * time.Minute // 关注关系
	TTLUnreadCount     = 0 * time.Second  // 不过期，手动失效
	TTLNotifCategories = 30 * time.Second // 通知分类未读数（高频接口短缓存）
	TTLViewHLL         = 25 * time.Hour   // HLL 浏览量去重 key（覆盖跨天边界）
	TTLRecommend       = 30 * time.Minute // 个性化推荐缓存（项目/日志/帖子/用户）
)

// ===========================
// Key 构造函数
// ===========================

// UserProfileKey 用户主页缓存 key
func UserProfileKey(userID uint64) string {
	return fmt.Sprintf(KeyUserProfile, userID)
}

// UserUnreadKey 未读通知数缓存 key
func UserUnreadKey(userID uint64) string {
	return fmt.Sprintf(KeyUserUnreadCnt, userID)
}

// UserNotifCategoriesKey 通知分类未读数缓存 key
func UserNotifCategoriesKey(userID uint64) string {
	return fmt.Sprintf(KeyUserNotifCats, userID)
}

// PostDetailKey 帖子详情缓存 key
func PostDetailKey(postID uint64) string {
	return fmt.Sprintf(KeyPostDetail, postID)
}

// PostHotListKey 热门帖子列表缓存 key
func PostHotListKey(limit int) string {
	return fmt.Sprintf("post:hot:list:%d", limit)
}

// ProjectDetailKey 项目详情缓存 key
func ProjectDetailKey(slug string) string {
	return fmt.Sprintf(KeyProjectDetail, slug)
}

// FollowRelationKey 关注关系缓存 key
func FollowRelationKey(followerID, followeeID uint64) string {
	return fmt.Sprintf(KeyFollowRelation, followerID, followeeID)
}

// LoginFailKey 登录失败计数 key（暴力破解防护）
func LoginFailKey(identifier string) string {
	return fmt.Sprintf(KeyLoginFailCnt, identifier)
}

// ===========================
// Feed 流
// ===========================

const (
	KeyUserFeed       = "feed:%d"      // %d = userID，ZSet，score = Unix 时间戳
	KeyUserFanoutLock = "feed:lock:%d" // %d = contentID，防止重复扇出
)

const (
	FanoutThreshold = 1000 // 粉丝数超过此值使用读扩散（大V）
	FeedMaxLen      = 500  // Feed ZSet 最大长度，超出时裁剪最老条目
)

// UserFeedKey 用户 Feed ZSet key
func UserFeedKey(userID uint64) string {
	return fmt.Sprintf(KeyUserFeed, userID)
}

// ===========================
// 浏览量 UV 去重（HyperLogLog）
// ===========================

const (
	KeyPostViewHLL = "view:post:%d:%s" // %d=postID, %s=YYYY-MM-DD
	KeyLogViewHLL  = "view:log:%d:%s"  // %d=logID, %s=YYYY-MM-DD
)

// PostViewHLLKey 帖子浏览 HLL key（当天）
func PostViewHLLKey(postID uint64, date string) string {
	return fmt.Sprintf(KeyPostViewHLL, postID, date)
}

// LogViewHLLKey 日志浏览 HLL key（当天）
func LogViewHLLKey(logID uint64, date string) string {
	return fmt.Sprintf(KeyLogViewHLL, logID, date)
}

// ===========================
// 个性化推荐缓存
// ===========================

// RecommendProjectsKey 推荐项目缓存（JSON []uint64）
func RecommendProjectsKey(userID uint64) string {
	return fmt.Sprintf("rec:proj:%d", userID)
}

// RecommendUsersKey 推荐用户缓存（JSON []uint64）
func RecommendUsersKey(userID uint64) string {
	return fmt.Sprintf("rec:user:%d", userID)
}

// RecommendLogsKey 推荐日志缓存（JSON []uint64）
func RecommendLogsKey(userID uint64) string {
	return fmt.Sprintf("rec:logs:%d", userID)
}

// RecommendPostsKey 推荐帖子缓存（JSON []uint64）
func RecommendPostsKey(userID uint64) string {
	return fmt.Sprintf("rec:posts:%d", userID)
}

// ===========================
// Deprecated aliases（兼容旧代码，逐步迁移到 TTLRecommend）
// ===========================

// RecommendTTL 已弃用，请使用 TTLRecommend
//
// Deprecated: use TTLRecommend
const RecommendTTL = TTLRecommend
