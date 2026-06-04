// ratelimit.go 提供基于 Redis 令牌桶算法的 HTTP 限流中间件。
// 令牌桶算法：每秒向桶中填充 rate 个令牌，桶容量为 burst。
// 每次请求消耗一个令牌，令牌不足时返回 429 Too Many Requests。
// 默认按客户端 IP 进行限流。
package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// tokenBucketScript 令牌桶 Lua 脚本
// KEYS[1]: 令牌桶 Redis key
// ARGV[1]: 填充速率（每秒令牌数）
// ARGV[2]: 桶容量（最大令牌数）
// ARGV[3]: 当前时间戳（纳秒）
// ARGV[4]: 请求消耗的令牌数（通常为 1）
// 返回：1 表示允许，0 表示拒绝
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local fill_time = capacity / rate
local ttl = math.floor(fill_time * 2)

local last_tokens = tonumber(redis.call("GET", key .. ":tokens"))
local last_refreshed = tonumber(redis.call("GET", key .. ":ts"))

if last_tokens == nil then
    last_tokens = capacity
end
if last_refreshed == nil then
    last_refreshed = 0
end

-- 计算经过时间（秒），添加的令牌数
local delta = math.max(0, (now - last_refreshed) / 1e9)
local filled_tokens = math.min(capacity, last_tokens + (delta * rate))
local allowed = filled_tokens >= requested

local new_tokens = filled_tokens
if allowed then
    new_tokens = filled_tokens - requested
end

redis.call("SETEX", key .. ":tokens", ttl, new_tokens)
redis.call("SETEX", key .. ":ts", ttl, now)

if allowed then
    return 1
else
    return 0
end
`)

// allowTokenBucket 执行令牌桶检查，返回是否允许通过。
// key: Redis key；rate: 每秒填充速率；burst: 桶容量
func allowTokenBucket(ctx context.Context, key string, rate float64, burst int) (bool, error) {
	rdb := cache.Get()
	if rdb == nil {
		return true, nil
	}
	now := time.Now().UnixNano()
	result, err := tokenBucketScript.Run(
		ctx,
		rdb,
		[]string{key},
		strconv.FormatFloat(rate, 'f', -1, 64),
		strconv.FormatFloat(float64(burst), 'f', -1, 64),
		fmt.Sprintf("%d", now),
		"1",
	).Int()
	if err != nil {
		return true, err // Redis 故障时放行
	}
	return result == 1, nil
}

// RateLimiter 限流中间件
// 根据配置对 HTTP 请求进行令牌桶限流
// keyFunc 用于从请求中提取限流 key（默认使用客户端 IP）
func RateLimiter(cfg config.RateLimitConfig) gin.HandlerFunc {
	return RateLimiterWithKey(cfg, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// RateLimiterWithKey 自定义 key 的限流中间件
// keyFunc 可以根据需要返回不同的 key，例如按用户 ID 或 API 路径限流
func RateLimiterWithKey(cfg config.RateLimitConfig, keyFunc func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		key := cfg.KeyPrefix + keyFunc(c)
		allowed, err := allowTokenBucket(c.Request.Context(), key, cfg.Rate, cfg.Burst)
		if err != nil {
			// Redis 出错时不阻断请求（降级处理）
			c.Next()
			return
		}

		if !allowed {
			// 添加 Retry-After 响应头
			c.Header("Retry-After", "1")
			response.FailTooManyRequests(c)
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkRateLimit 执行令牌桶限流检查（保留向后兼容）
func checkRateLimit(ctx context.Context, key string, rate, capacity float64) (bool, error) {
	return allowTokenBucket(ctx, key, rate, int(capacity))
}

// RateLimiterByUserID 按用户 ID 限流的中间件
// 已登录用户按 user_id 限流，未登录用户按 IP 限流
func RateLimiterByUserID(cfg config.RateLimitConfig) gin.HandlerFunc {
	return RateLimiterWithKey(cfg, func(c *gin.Context) string {
		if userID, ok := GetUserID(c); ok {
			return fmt.Sprintf("user:%d", userID)
		}
		return "ip:" + c.ClientIP()
	})
}

// RateLimiterByPath 按 API 路径+IP 限流的中间件
func RateLimiterByPath(cfg config.RateLimitConfig) gin.HandlerFunc {
	return RateLimiterWithKey(cfg, func(c *gin.Context) string {
		return fmt.Sprintf("path:%s:%s", c.FullPath(), c.ClientIP())
	})
}

// RateLimiterByUser 对已认证用户按 UserID 限流，未认证时按 IP 限流。
// 已登录用户享有 3 倍速率，鼓励正常使用。
func RateLimiterByUser(cfg config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		var key string
		rate := cfg.Rate
		burst := cfg.Burst
		if uid, exists := c.Get(UserIDKey); exists && uid != nil {
			key = fmt.Sprintf("%suser:%v", cfg.KeyPrefix, uid)
			rate = cfg.Rate * 3
			burst = cfg.Burst * 3
		} else {
			key = fmt.Sprintf("%sip:%s", cfg.KeyPrefix, c.ClientIP())
		}
		allowed, err := allowTokenBucket(c.Request.Context(), key, rate, burst)
		if err != nil {
			logger.Warn("ratelimit: Redis error, allowing request", zap.Error(err))
			c.Next()
			return
		}
		if !allowed {
			response.FailTooManyRequests(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// StrictRateLimiter 针对高危接口（发验证码/登录/注册）的严格 IP 限流。
// 默认：每 IP 每分钟最多 5 次（rate=5/60≈0.083, burst=5）。
// 触发时返回 429，并设置 Retry-After header（秒）。
func StrictRateLimiter(cfg config.RateLimitConfig) gin.HandlerFunc {
	strict := config.RateLimitConfig{
		Enabled:   cfg.Enabled,
		Rate:      5.0 / 60,  // 每分钟 5 次
		Burst:     5,
		KeyPrefix: "strict:" + cfg.KeyPrefix,
	}
	return RateLimiterWithKey(strict, func(c *gin.Context) string {
		return "ip:" + c.ClientIP()
	})
}
