// Package modules 提供路由模块化拆分的各模块实现。
// infra.go：基础设施与依赖初始化。
package modules

import (
	"context"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/internal/ws"
	"github.com/gamero/gamero/pkg/database"
	"github.com/gamero/gamero/pkg/filter"
	"github.com/gamero/gamero/pkg/logger"
	"github.com/gamero/gamero/pkg/mq"
	"github.com/gamero/gamero/pkg/search"
	"github.com/gamero/gamero/pkg/storage"
	"go.uber.org/zap"
)

// InitDeps 初始化所有共享依赖，返回填充好的 *Deps
func InitDeps(ctx context.Context, cfg *config.Config) *Deps {
	log := logger.Get()

	// ========== 数据库 ==========
	db := database.Get()

	// ========== 存储 ==========
	stor := storage.Get()

	// ========== 消息队列 Broker ==========
	var broker mq.Broker
	switch cfg.MQ.Provider {
	case "kafka":
		batchTimeout := time.Duration(cfg.MQ.BatchTimeout) * time.Millisecond
		if batchTimeout <= 0 {
			batchTimeout = 100 * time.Millisecond
		}
		maxWait := time.Duration(cfg.MQ.MaxWaitMs) * time.Millisecond
		if maxWait <= 0 {
			maxWait = 500 * time.Millisecond
		}
		commitInterval := time.Duration(cfg.MQ.CommitInterval) * time.Millisecond
		if commitInterval <= 0 {
			commitInterval = time.Second
		}
		maxBytes := cfg.MQ.MaxBytes
		if maxBytes <= 0 {
			maxBytes = 10 << 20 // 10 MB
		}
		broker = mq.NewKafkaBrokerWithConfig(mq.KafkaBrokerConfig{
			Brokers:        cfg.MQ.Brokers,
			GroupID:        cfg.MQ.GroupID,
			TopicPrefix:    cfg.MQ.TopicPrefix,
			BatchSize:      cfg.MQ.BatchSize,
			BatchTimeout:   batchTimeout,
			MinBytes:       cfg.MQ.MinBytes,
			MaxBytes:       maxBytes,
			MaxWait:        maxWait,
			CommitInterval: commitInterval,
			Logger:         log,
		})
	default:
		bufSize := cfg.MQ.BufSize
		if bufSize <= 0 {
			bufSize = 1000
		}
		broker = mq.NewMemoryBroker(bufSize, log)
	}
	// 启动消费循环
	broker.Start(context.Background())

	// ========== 敏感词过滤器（DB 优先，再合并默认词库）==========
	{
		swRepo := repository.NewSensitiveWordRepository(db)
		dbWords, err := swRepo.LoadAll(context.Background())
		if err != nil {
			log.Warn("敏感词 DB 加载失败，使用默认词库", zap.Error(err))
			dbWords = nil
		}
		// 将默认词库与 DB 词库合并，去重后初始化
		merged := filter.DefaultSensitiveWords
		if len(dbWords) > 0 {
			seen := make(map[string]struct{}, len(merged))
			for _, w := range merged {
				seen[w] = struct{}{}
			}
			for _, w := range dbWords {
				if _, ok := seen[w]; !ok {
					merged = append(merged, w)
				}
			}
		}
		filter.Init(merged)
		log.Info("敏感词过滤器已加载",
			zap.Int("default", len(filter.DefaultSensitiveWords)),
			zap.Int("from_db", len(dbWords)),
			zap.Int("total", len(merged)),
		)
	}

	// ========== WebSocket Hub ==========
	wsHub := ws.NewHub(log)
	go wsHub.StartPubSub(ctx)

	// ========== 搜索器 ==========
	searcher := initSearcher(cfg, log)

	// ========== 共享 Repo ==========
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	followRepo := repository.NewFollowRepository(db)
	reportRepo := repository.NewReportRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	devLogRepo := repository.NewDevLogRepository(db)

		// ========== NotificationPreferenceRepo ==========
		notifPrefRepo := repository.NewNotificationPreferenceRepository(db)


	// ========== 共享 Svc ==========
	userSvc := service.NewUserService(userRepo, teamRepo)

	// ========== NotifSvc ==========
	notifSvc := service.NewNotificationService(teamRepo, wsHub)

	// ========== ReviewSvc ==========
	reviewRepo := repository.NewReviewRepository(db)
	reviewSvc := service.NewReviewService(reviewRepo, userRepo, projectRepo)

	// ========== FeedSvc ==========
	feedSvc := service.NewFeedService(
		followRepo,
		userRepo,
		projectRepo,
		devLogRepo,
		repository.NewPostRepository(db),
		stor,
	)

	// ========== ModerationSvc ==========
	postRepo := repository.NewPostRepository(db)
	moderationSvc := service.NewModerationService(
		userRepo,
		projectRepo,
		devLogRepo,
		postRepo,
		notifSvc,
		cfg.Moderation.CallbackSecret,
		cfg.Moderation.AllowedCallbackIPs,
	)

	return &Deps{
		Ctx:         ctx,
		Cfg:         cfg,
		DB:          db,
		Log:         log,
		Stor:        stor,
		Broker:      broker,
		Searcher:    searcher,
		WSHub:       wsHub,
		UserRepo:    userRepo,
		TeamRepo:    teamRepo,
		ProjectRepo: projectRepo,
		FollowRepo:  followRepo,
		ReportRepo:  reportRepo,
		AdminRepo:   adminRepo,
		DevLogRepo:  devLogRepo,
		PostRepo:    postRepo,
			NotifPrefRepo: notifPrefRepo,
		UserSvc:     userSvc,
		NotifSvc:     notifSvc, // 统一通知服务
		ModerationSvc: moderationSvc,
		ReviewRepo:   reviewRepo,
		ReviewSvc:    reviewSvc,
		FeedSvc:      feedSvc,
	}
}

// initSearcher 根据配置初始化搜索器
func initSearcher(cfg *config.Config, log *zap.Logger) search.Searcher {
	if cfg.Elasticsearch.Enabled {
		es, err := search.NewESSearcher(
			cfg.Elasticsearch.Addresses,
			cfg.Elasticsearch.Username,
			cfg.Elasticsearch.Password,
			cfg.Elasticsearch.IndexPrefix,
			log,
		)
		if err != nil {
			log.Warn("ES 初始化失败，降级到 PostgreSQL FTS", zap.Error(err))
			return search.NewNoopSearcher()
		}
		log.Info("搜索引擎：Elasticsearch", zap.Strings("addrs", cfg.Elasticsearch.Addresses))
		return es
	}
	log.Info("搜索引擎：PostgreSQL FTS（ES 未启用）")
	return search.NewNoopSearcher()
}
