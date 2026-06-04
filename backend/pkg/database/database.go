// Package database 提供 PostgreSQL 数据库连接和 GORM ORM 封装。
// 支持连接池配置、自动迁移，并对外暴露 *gorm.DB 实例。
// 当 DatabaseConfig.ReplicaDSNs 非空时，自动启用读写分离：
//   - SELECT 查询路由到只读副本（随机策略）
//   - INSERT / UPDATE / DELETE / 事务内所有操作路由到主库
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
)

// global 是全局数据库实例
var global *gorm.DB

// Init 根据配置初始化数据库连接
func Init(cfg config.DatabaseConfig) (*gorm.DB, error) {
	// 配置 GORM 日志级别
	gormLog := newGormLogger()

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormLog,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false, // 使用复数表名
		},
		PrepareStmt:          true,  // 开启预编译语句缓存
		DisableAutomaticPing: false, // 启动时自动 Ping
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层 *sql.DB 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxLifetime/2) * time.Second)

	// 验证连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库 Ping 失败: %w", err)
	}

	// 读写分离：若配置了只读副本，注册 DBResolver
	if len(cfg.ReplicaDSNs) > 0 {
		replicas := make([]gorm.Dialector, 0, len(cfg.ReplicaDSNs))
		for _, dsn := range cfg.ReplicaDSNs {
			replicas = append(replicas, postgres.Open(dsn))
		}

		if err := db.Use(dbresolver.Register(dbresolver.Config{
			// Sources（主库）使用默认连接，Replicas（从库）用副本 DSN
			Replicas: replicas,
			// 负载均衡策略：随机
			Policy: dbresolver.RandomPolicy{},
		}).
			SetMaxIdleConns(cfg.MaxIdleConns).
			SetMaxOpenConns(cfg.MaxOpenConns).
			SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)); err != nil {
			return nil, fmt.Errorf("读写分离初始化失败: %w", err)
		}

		logger.Info("读写分离已启用", zap.Int("replica_count", len(cfg.ReplicaDSNs)))
	}

	global = db
	logger.Info("数据库连接成功",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("dbname", cfg.DBName),
	)

	return db, nil
}

// Get 返回全局数据库实例
func Get() *gorm.DB {
	if global == nil {
		panic("数据库未初始化，请先调用 database.Init()")
	}
	return global
}

// AutoMigrate 自动迁移数据模型到数据库
// models 参数为要迁移的模型实例列表（指针类型）
func AutoMigrate(models ...interface{}) error {
	db := Get()
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("自动迁移失败: %w", err)
	}
	logger.Info("数据库自动迁移完成", zap.Int("models", len(models)))
	return nil
}

// Ping 检查数据库连接状态
func Ping() error {
	db := Get()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Close 关闭数据库连接
func Close() error {
	if global == nil {
		return nil
	}
	sqlDB, err := global.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Transaction 在事务中执行函数
func Transaction(fn func(tx *gorm.DB) error) error {
	return Get().Transaction(fn)
}

// gormZapLogger 实现 gorm logger.Interface，将日志输出到 zap
type gormZapLogger struct {
	SlowThreshold time.Duration
}

// newGormLogger 创建 GORM 的 zap 日志适配器
func newGormLogger() gormlogger.Interface {
	return &gormZapLogger{
		SlowThreshold: 200 * time.Millisecond,
	}
}

// LogMode 设置日志级别（实现 gormlogger.Interface）
func (l *gormZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

// Info 记录 Info 日志
func (l *gormZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	logger.Info(fmt.Sprintf(msg, data...))
}

// Warn 记录 Warn 日志
func (l *gormZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	logger.Warn(fmt.Sprintf(msg, data...))
}

// Error 记录 Error 日志
func (l *gormZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	logger.Error(fmt.Sprintf(msg, data...))
}

// Trace 记录 SQL 执行日志
func (l *gormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	if err != nil && err != gorm.ErrRecordNotFound {
		fields = append(fields, zap.Error(err))
		logger.Error("SQL 执行错误", fields...)
		return
	}

	if elapsed > l.SlowThreshold {
		logger.Warn("慢 SQL 警告", fields...)
		return
	}

	logger.Debug("SQL 执行", fields...)
}
