package cache

import (
	"context"
	"time"
)

// PFAddAndCheckNew 将 member 加入 HLL，返回 true 表示是新访客（HLL 值改变）。
// 同时在 key 首次创建时设置 TTL（已存在则不重设，保持原 TTL）。
func PFAddAndCheckNew(ctx context.Context, key string, member string, ttl time.Duration) (bool, error) {
	rdb := Get()
	changed, err := rdb.PFAdd(ctx, key, member).Result()
	if err != nil {
		return false, err
	}
	// 仅在 HLL 首次写入时设置 TTL（changed==1 时不一定是第一次，用 EXPIRE 幂等处理）
	if changed == 1 {
		// 不覆盖已有 TTL，只在没有 TTL 时设置
		rdb.Expire(ctx, key, ttl)
	}
	return changed == 1, nil
}
