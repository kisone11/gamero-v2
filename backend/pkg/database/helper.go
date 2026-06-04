package database

import (
	"context"

	"gorm.io/gorm"
)

// Write 返回用于写操作的 DB 实例（强制走主库）
// gorm dbresolver 会根据操作类型自动路由：INSERT/UPDATE/DELETE/事务 → 主库。
// 此函数主要提供语义化封装，方便将来扩展（如可加 dbresolver.Write 子句强制走主库）。
func Write(ctx context.Context) *gorm.DB {
	return Get().WithContext(ctx)
}

// Read 返回用于读操作的 DB 实例（走从库，有副本时自动路由）
// gorm dbresolver 会根据操作类型自动路由：SELECT → 从库（随机负载均衡）。
func Read(ctx context.Context) *gorm.DB {
	return Get().WithContext(ctx)
}
