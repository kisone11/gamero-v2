// main.go 是 gamero 服务的程序入口。
// 负责初始化配置、各组件（数据库、Redis、对象存储、JWT 等），启动 HTTP 服务器。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/internal/router"
	"github.com/gamero/gamero/pkg/auth"
	"github.com/gamero/gamero/pkg/cache"
	"github.com/gamero/gamero/pkg/database"
	"github.com/gamero/gamero/pkg/email"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/sms"
	"github.com/gamero/gamero/pkg/storage"
	"go.uber.org/zap"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志系统（最先初始化，其他组件依赖日志）
	if err := logger.Init(cfg.Logger); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Gamero 服务启动中...",
		zap.String("mode", cfg.Server.Mode),
		zap.Int("port", cfg.Server.Port),
	)

	// 初始化数据库
	if _, err := database.Init(cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Error("关闭数据库连接失败", zap.Error(err))
		}
	}()

	// 初始化 Redis
	if _, err := cache.Init(cfg.Redis); err != nil {
		logger.Fatal("初始化 Redis 失败", zap.Error(err))
	}
	defer func() {
		if err := cache.Close(); err != nil {
			logger.Error("关闭 Redis 连接失败", zap.Error(err))
		}
	}()

	// 初始化对象存储
	if _, err := storage.Init(cfg.Storage); err != nil {
		logger.Fatal("初始化对象存储失败", zap.Error(err))
	}

	// 初始化 JWT 管理器
	auth.Init(cfg.JWT)

	// 初始化短信服务
	sms.Init(cfg.SMS)

	// 初始化邮件服务
	email.Init(cfg.Email)

	// 创建可取消的 context，用于优雅关闭后台任务
	appCtx, appCancel := context.WithCancel(context.Background())

	// 初始化路由（传入 appCtx，后台 goroutine 会随 ctx 取消而退出）
	engine := router.Setup(appCtx, cfg)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        engine,
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// 在后台启动 HTTP 服务器
	go func() {
		logger.Info("HTTP 服务器已启动", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("HTTP 服务器启动失败", zap.Error(err))
		}
	}()

	// 等待终止信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在优雅关闭服务器...")

	// 先取消 appCtx，通知所有后台 goroutine 退出
	appCancel()

	// 优雅关闭 HTTP 服务器：最多等待 30 秒
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务器强制关闭", zap.Error(err))
	}

	logger.Info("服务器已关闭")
}
