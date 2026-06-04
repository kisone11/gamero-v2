// Package modules 提供路由模块化拆分的共享依赖定义。
package modules

import (
	"context"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/internal/service"
	"github.com/gamero/gamero/internal/ws"
	"github.com/gamero/gamero/pkg/mq"
	"github.com/gamero/gamero/pkg/search"
	"github.com/gamero/gamero/pkg/storage"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Deps 模块间共享的基础依赖(由 InitDeps 在 Setup 入口初始化)
type Deps struct {
	Ctx      context.Context
	Cfg      *config.Config
	DB       *gorm.DB
	Log      *zap.Logger
	Stor     *storage.Client
	Broker   mq.Broker
	Searcher search.Searcher
	WSHub    *ws.Hub

	// 跨模块共享的 repo
	UserRepo    repository.UserRepository
	TeamRepo    repository.TeamRepository
	ProjectRepo repository.ProjectRepository
	FollowRepo  repository.FollowRepository
	ReportRepo  repository.ReportRepository
	AdminRepo   repository.AdminRepository
	NotifPrefRepo    repository.NotificationPreferenceRepository
	DevLogRepo  repository.DevLogRepository
	PostRepo    repository.PostRepository


	// 跨模块共享的 svc
	UserSvc      service.UserService
	CommunitySvc service.CommunityService
	ProjectSvc   service.ProjectService
	DevLogSvc    service.DevLogService
	TeamSvc      service.TeamService
	FollowSvc    service.FollowService
	StatsSvc     service.StatsService
	NotifSvc     service.NotificationService  // 统一通知服务
	ModerationSvc service.ModerationService   // 内容审核服务
	ReviewRepo   repository.ReviewRepository  // 评测仓库
	ReviewSvc    service.ReviewService        // 评测服务
	FeedSvc      service.FeedService
}
