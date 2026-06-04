// Package cache 提供基于 go-redis v9 的 Redis 连接和常用操作封装。
// 支持连接池配置，提供 String、Hash、List、ZSet 等常用数据结构操作。
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// global 是全局 Redis 客户端实例
var global *redis.Client

// Init 根据配置初始化 Redis 连接
func Init(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}

	global = client
	logger.Info("Redis 连接成功",
		zap.String("addr", cfg.Addr()),
		zap.Int("db", cfg.DB),
	)

	return client, nil
}

// Get 返回全局 Redis 客户端。
// 若 Redis 未初始化则返回 nil（调用方需检查 nil 后再使用）。
func Get() *redis.Client {
	return global
}

// Ping 检查 Redis 连接状态
func Ping() error {
	if global == nil {
		return errors.New("redis not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return Get().Ping(ctx).Err()
}

// Close 关闭 Redis 连接
func Close() error {
	if global == nil {
		return nil
	}
	return global.Close()
}

// InitForTest 使用指定 addr 初始化一个无密码的 Redis 客户端（仅供单元测试使用）
func InitForTest(addr string) {
	global = redis.NewClient(&redis.Options{
		Addr: addr,
	})
}

// ===== String 操作 =====

// Set 设置 key-value，ttl=0 表示不过期
func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return Get().Set(ctx, key, value, ttl).Err()
}

// SetJSON 将 value 序列化为 JSON 后存储到 Redis
func SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache.SetJSON: marshal failed: %w", err)
	}
	return Get().Set(ctx, key, data, ttl).Err()
}

// GetJSON 从 Redis 读取并反序列化 JSON 到 dest。
// 返回 (true, nil) 表示命中；(false, nil) 表示 miss；(false, err) 表示错误。
func GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := Get().Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("cache.GetJSON: unmarshal failed: %w", err)
	}
	return true, nil
}

// SetNX 仅当 key 不存在时设置，返回是否设置成功
func SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return Get().SetNX(ctx, key, value, ttl).Result()
}

// GetString 获取 string 类型的值
func GetString(ctx context.Context, key string) (string, error) {
	return Get().Get(ctx, key).Result()
}

// Del 删除一个或多个 key
func Del(ctx context.Context, keys ...string) error {
	return Get().Del(ctx, keys...).Err()
}

// Exists 检查 key 是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	n, err := Get().Exists(ctx, key).Result()
	return n > 0, err
}

// Expire 设置 key 的过期时间
func Expire(ctx context.Context, key string, ttl time.Duration) error {
	return Get().Expire(ctx, key, ttl).Err()
}

// TTL 获取 key 的剩余过期时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	return Get().TTL(ctx, key).Result()
}

// Incr 对 key 的整数值加 1
func Incr(ctx context.Context, key string) (int64, error) {
	return Get().Incr(ctx, key).Result()
}

// IncrBy 对 key 的整数值加 n
func IncrBy(ctx context.Context, key string, n int64) (int64, error) {
	return Get().IncrBy(ctx, key, n).Result()
}

// ===== Hash 操作 =====

// HSet 设置 Hash 字段值，fields 为 field-value 交替对
func HSet(ctx context.Context, key string, fields ...interface{}) error {
	return Get().HSet(ctx, key, fields...).Err()
}

// HGet 获取 Hash 单个字段值
func HGet(ctx context.Context, key, field string) (string, error) {
	return Get().HGet(ctx, key, field).Result()
}

// HGetAll 获取 Hash 所有字段和值
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return Get().HGetAll(ctx, key).Result()
}

// HDel 删除 Hash 中的字段
func HDel(ctx context.Context, key string, fields ...string) error {
	return Get().HDel(ctx, key, fields...).Err()
}

// HExists 检查 Hash 字段是否存在
func HExists(ctx context.Context, key, field string) (bool, error) {
	return Get().HExists(ctx, key, field).Result()
}

// HIncrBy 对 Hash 字段整数值加 n
func HIncrBy(ctx context.Context, key, field string, n int64) (int64, error) {
	return Get().HIncrBy(ctx, key, field, n).Result()
}

// ===== List 操作 =====

// LPush 从左侧推入列表元素
func LPush(ctx context.Context, key string, values ...interface{}) error {
	return Get().LPush(ctx, key, values...).Err()
}

// RPush 从右侧推入列表元素
func RPush(ctx context.Context, key string, values ...interface{}) error {
	return Get().RPush(ctx, key, values...).Err()
}

// LPop 从左侧弹出列表元素
func LPop(ctx context.Context, key string) (string, error) {
	return Get().LPop(ctx, key).Result()
}

// RPop 从右侧弹出列表元素
func RPop(ctx context.Context, key string) (string, error) {
	return Get().RPop(ctx, key).Result()
}

// LRange 获取列表范围元素（0 到 -1 表示全部）
func LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return Get().LRange(ctx, key, start, stop).Result()
}

// LLen 获取列表长度
func LLen(ctx context.Context, key string) (int64, error) {
	return Get().LLen(ctx, key).Result()
}

// ===== Sorted Set 操作 =====

// ZAdd 添加有序集合成员
func ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return Get().ZAdd(ctx, key, members...).Err()
}

// ZRange 按分数升序获取有序集合成员（0 到 -1 表示全部）
func ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return Get().ZRange(ctx, key, start, stop).Result()
}

// ZRevRange 按分数降序获取有序集合成员
func ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return Get().ZRevRange(ctx, key, start, stop).Result()
}

// ZRangeWithScores 按分数升序获取有序集合成员及分数
func ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return Get().ZRangeWithScores(ctx, key, start, stop).Result()
}

// ZRangeByScore 按分数区间获取有序集合成员
func ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return Get().ZRangeByScore(ctx, key, opt).Result()
}

// ZRem 删除有序集合成员
func ZRem(ctx context.Context, key string, members ...interface{}) error {
	return Get().ZRem(ctx, key, members...).Err()
}

// ZScore 获取有序集合成员的分数
func ZScore(ctx context.Context, key, member string) (float64, error) {
	return Get().ZScore(ctx, key, member).Result()
}

// ZCard 获取有序集合成员数量
func ZCard(ctx context.Context, key string) (int64, error) {
	return Get().ZCard(ctx, key).Result()
}

// ZRank 获取成员在有序集合中的排名（升序，从 0 开始）
func ZRank(ctx context.Context, key, member string) (int64, error) {
	return Get().ZRank(ctx, key, member).Result()
}

// ZRevRank 获取成员在有序集合中的降序排名（从 0 开始）
func ZRevRank(ctx context.Context, key, member string) (int64, error) {
	return Get().ZRevRank(ctx, key, member).Result()
}

// ===== Set 操作 =====

// SAdd 添加集合成员
func SAdd(ctx context.Context, key string, members ...interface{}) error {
	return Get().SAdd(ctx, key, members...).Err()
}

// SIsMember 检查成员是否在集合中
func SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return Get().SIsMember(ctx, key, member).Result()
}

// SRem 删除集合成员
func SRem(ctx context.Context, key string, members ...interface{}) error {
	return Get().SRem(ctx, key, members...).Err()
}

// SCard 获取集合成员数量
func SCard(ctx context.Context, key string) (int64, error) {
	return Get().SCard(ctx, key).Result()
}

// SMembers 获取集合所有成员
func SMembers(ctx context.Context, key string) ([]string, error) {
	return Get().SMembers(ctx, key).Result()
}

// ===== 通用操作 =====

// Keys 获取匹配 pattern 的所有 key（生产环境慎用，推荐 SCAN）
func Keys(ctx context.Context, pattern string) ([]string, error) {
	return Get().Keys(ctx, pattern).Result()
}

// Scan 游标扫描 key（替代 KEYS，生产环境推荐）
func Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return Get().Scan(ctx, cursor, match, count).Result()
}

// Pipeline 返回 Redis Pipeline，用于批量操作
func Pipeline() redis.Pipeliner {
	return Get().Pipeline()
}

// TxPipeline 返回事务 Pipeline
func TxPipeline() redis.Pipeliner {
	return Get().TxPipeline()
}
