// tasks.go：后台定时任务（随 ctx 生命周期运行）。
package modules

import (
	"context"
	"time"

	"github.com/gamero/gamero/pkg/cache"
	"go.uber.org/zap"
)

// StartBackgroundTasks 启动所有后台定时 goroutine。
// ctx 取消时所有 goroutine 自动退出，实现优雅关闭。
// 需在所有 svc 都已挂载到 deps 后调用（RegisterXxxRoutes 之后）。
func StartBackgroundTasks(ctx context.Context, deps *Deps) {
	log := deps.Log

	// 每小时自动关闭过期招募帖
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count, err := deps.TeamSvc.CloseExpiredRecruitments(context.Background())
				if err == nil && count > 0 {
					log.Info("自动关闭过期招募", zap.Int64("count", count))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 每小时自动将超过 24h 未处理的 pending 举报升级为 escalated
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count, err := deps.ReportRepo.EscalatePendingReports(context.Background(), 24*time.Hour)
				if err != nil {
					log.Error("举报自动升级失败", zap.Error(err))
				} else if count > 0 {
					log.Info("举报自动升级", zap.Int64("count", count))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 每 30 分钟自动解封到期用户
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count, err := deps.AdminRepo.UnbanExpiredUsers(context.Background())
				if err != nil {
					log.Error("自动解封用户失败", zap.Error(err))
				} else if count > 0 {
					log.Info("自动解封到期用户", zap.Int64("count", count))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 每小时全量刷新帖子热度分数（多实例通过 Redis 分布式锁保证幂等）
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if deps.CommunitySvc == nil {
					continue
				}
				const lockKey = "lock:hot_score_refresh_full"
				const lockTTL = 54 * time.Minute
				acquired, err := cache.SetNX(context.Background(), lockKey, "1", lockTTL)
				if err != nil || !acquired {
					continue
				}
				if err := deps.CommunitySvc.RefreshHotScores(context.Background()); err != nil {
					log.Error("帖子热度刷新失败", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 每 10 分钟精准刷新最近 7 天有互动帖子的热度分数
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if deps.CommunitySvc == nil {
					continue
				}
				const lockKey = "lock:hot_score_refresh"
				const lockTTL = 9 * time.Minute
				acquired, err := cache.SetNX(ctx, lockKey, "1", lockTTL)
				if err != nil || !acquired {
					continue
				}
				if err := deps.CommunitySvc.RefreshRecentHotScores(ctx); err != nil {
					log.Error("帖子热度精准刷新失败", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 每日凌晨 2 点聚合统计数据
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
			delay := time.NewTimer(time.Until(next))
			select {
			case <-delay.C:
				if deps.StatsSvc != nil {
					yesterday := time.Now().AddDate(0, 0, -1)
					if err := deps.StatsSvc.RunDailyAggregation(ctx, yesterday); err != nil {
						log.Error("daily stats aggregation failed", zap.Error(err))
					}
				}
			case <-ctx.Done():
				delay.Stop()
				return
			}
		}
	}()
}
