// Package router 提供 HTTP 路由注册入口。
// 所有模块路由通过 internal/router/modules/ 下的独立文件注册。
package router

import (
	"context"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/internal/handler"
	"github.com/gamero/gamero/internal/router/modules"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/metrics"
	"github.com/gamero/gamero/pkg/middleware"
	"github.com/gamero/gamero/pkg/tracing"
	"github.com/gamero/gamero/pkg/validator"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelgin "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

// Setup 初始化并返回 Gin Engine，注册所有路由和中间件。
//
// ctx 用于控制后台定时 goroutine 的生命周期：当 ctx 被取消时（通常在 main 收到 SIGTERM 后），
// 所有后台 goroutine 会自动退出，实现优雅关闭。
func Setup(ctx context.Context, cfg *config.Config) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)

	log := logger.Get()

	// ========== 注册自定义验证器 ==========
	if err := validator.RegisterCustomValidators(); err != nil {
		log.Warn("注册自定义验证器失败", zap.Error(err))
	}

	// ========== 链路追踪初始化 ==========
	tracingCfg := tracing.TracerConfig{
		Enabled:     cfg.Observability.Tracing.Enabled,
		ServiceName: cfg.Observability.Tracing.ServiceName,
		SampleRate:  cfg.Observability.Tracing.SampleRate,
		Exporter:    cfg.Observability.Tracing.Exporter,
		Endpoint:    cfg.Observability.Tracing.Endpoint,
	}
	if _, err := tracing.Init(tracingCfg, log); err != nil {
		log.Warn("链路追踪初始化失败", zap.Error(err))
	}

	engine := gin.New()

	// 限制 multipart 内存缓冲区（默认 32MB 过大）——文件上传路由另有 size 校验，这里只限 multipart form 解析阶段内存占用
	engine.MaxMultipartMemory = 11 << 20 // 11MB：最大上传 10MB + 1MB header

	// ========== 全局中间件 ==========
	engine.Use(middleware.Recovery())
	engine.Use(middleware.RequestLogger())
	if len(cfg.Server.AllowOrigins) > 0 {
		corsCfg := middleware.DefaultCORSConfig()
		corsCfg.AllowOrigins = cfg.Server.AllowOrigins
		engine.Use(middleware.CORSWithConfig(corsCfg))
	} else {
		engine.Use(middleware.CORS())
	}
	engine.Use(middleware.SecurityHeaders())
	engine.Use(middleware.MaxBodySize())
	engine.Use(middleware.RateLimiter(cfg.RateLimit))
	engine.Use(metrics.PrometheusMiddleware())
	if cfg.Observability.Tracing.Enabled {
		engine.Use(otelgin.Middleware(cfg.Observability.Tracing.ServiceName))
	}

	// ========== 系统端点 ==========
	engine.GET("/health", handler.Health)
	if cfg.Observability.Metrics.Enabled {
		metricsPath := cfg.Observability.Metrics.Path
		if metricsPath == "" {
			metricsPath = "/metrics"
		}
		engine.GET(metricsPath, gin.WrapH(promhttp.Handler()))
		log.Info("Prometheus 指标端点已注册", zap.String("path", metricsPath))
	}

	// ========== 初始化共享依赖 ==========
	deps := modules.InitDeps(ctx, cfg)

	// ========== 注册各模块路由 ==========
	v1 := engine.Group("/api/v1")

	// 注意：模块注册顺序决定了 svc 挂载顺序，community 依赖 project svc，需在其之后注册
	modules.RegisterUserRoutes(v1, deps)
	modules.RegisterProjectRoutes(v1, deps) // → deps.ProjectSvc
	modules.RegisterTeamRoutes(v1, deps)    // → deps.TeamSvc
	modules.RegisterDevLogRoutes(v1, deps)  // → deps.DevLogSvc
	modules.RegisterFollowRoutes(v1, deps)  // → deps.FollowSvc
	modules.RegisterFeedRoutes(v1, deps)    // deps.FeedSvc
	// community 须在 project 之后注册：内部会调用 deps.ProjectSvc.SetBroker + mq.RegisterSubscribers
	modules.RegisterCommunityRoutes(v1, deps) // → deps.CommunitySvc
	modules.RegisterDiscoverRoutes(v1, deps)
	modules.RegisterReviewRoutes(v1, deps)
	modules.RegisterStatsRoutes(v1, deps) // → deps.StatsSvc
	modules.RegisterAdminRoutes(v1, deps)
	modules.RegisterAnnouncementRoutes(v1, deps)
	modules.RegisterUploadRoutes(v1, deps)     // 前端直传云存储凭证
	modules.RegisterModerationRoutes(v1, deps) // 腾讯云 COS 审核回调

	// ========== 启动后台定时任务 ==========
	// 所有 svc 完成挂载后启动，保证 deps 字段已填充
	modules.StartBackgroundTasks(ctx, deps)

	return engine
}
