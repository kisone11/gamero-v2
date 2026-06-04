// Package service 提供创作者数据统计的业务逻辑层。
// 包含以下功能：
//   - GetMyStats：获取当前创作者的内容互动/收入/粉丝统计
//   - GetProjectStats：获取指定项目的每日统计
//   - RunDailyAggregation：定时聚合任务入口（从各业务表汇总写入 daily_stats）
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	apperrors "github.com/gamero/gamero/pkg/errors"
)

// StatsService 创作者统计服务接口
type StatsService interface {
	// GetMyStats 获取当前用户的统计数据
	GetMyStats(ctx context.Context, userID uint64, days int) (*StatsResponse, error)
	// GetProjectStats 获取项目统计（仅 project owner 可调用）
	GetProjectStats(ctx context.Context, projectID, requesterID uint64, days int) (*ProjectStatsResponse, error)
	// RunDailyAggregation 手动触发每日聚合（定时任务调用）
	RunDailyAggregation(ctx context.Context, date time.Time) error
}

// StatsResponse 创作者统计响应
type StatsResponse struct {
	Total  *repository.CreatorTotalStats `json:"total"`
	Daily  []*model.DailyStats           `json:"daily"`
	Period string                        `json:"period"` // "7d" | "30d" | "90d"
}

// ProjectSummaryStats 项目汇总统计（全时直实时查询）
type ProjectSummaryStats = repository.ProjectSummaryStats

// ProjectStatsResponse 项目统计响应
type ProjectStatsResponse struct {
	ProjectID uint64                `json:"project_id"`
	Summary   *ProjectSummaryStats  `json:"summary"`
	Daily     []*model.ProjectStats `json:"daily"`
	Period    string                `json:"period"`
}

type statsService struct {
	statsRepo   repository.StatsRepository
	userRepo    repository.UserRepository
	projectRepo repository.ProjectRepository
}

// NewStatsService 创建 StatsService 实例
func NewStatsService(statsRepo repository.StatsRepository, userRepo repository.UserRepository, projectRepo repository.ProjectRepository) StatsService {
	return &statsService{
		statsRepo:   statsRepo,
		userRepo:    userRepo,
		projectRepo: projectRepo,
	}
}

func (s *statsService) GetMyStats(ctx context.Context, userID uint64, days int) (*StatsResponse, error) {
	if days <= 0 || days > 90 {
		days = 30
	}
	to := time.Now()
	from := to.AddDate(0, 0, -days)

	daily, err := s.statsRepo.GetUserDailyStats(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	// 补零填充：将查询结果映射到完整日期区间，缺失日期用零值填充，保证前端折线图连续
	daily = fillDailyStats(userID, daily, from, to)

	total, err := s.statsRepo.GetUserTotalStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	period := fmt.Sprintf("%dd", days)

	return &StatsResponse{
		Total:  total,
		Daily:  daily,
		Period: period,
	}, nil
}

// fillDailyStats 对 [from, to] 区间内每天进行零值填充，缺失的天补入空记录
func fillDailyStats(userID uint64, existing []*model.DailyStats, from, to time.Time) []*model.DailyStats {
	// 建立日期 → 记录 map
	byDate := make(map[string]*model.DailyStats, len(existing))
	for _, s := range existing {
		byDate[s.Date.Format("2006-01-02")] = s
	}

	// 按日期顺序填充
	result := make([]*model.DailyStats, 0, int(to.Sub(from).Hours()/24)+1)
	for d := truncateToDay(from); !d.After(truncateToDay(to)); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if rec, ok := byDate[key]; ok {
			result = append(result, rec)
		} else {
			result = append(result, &model.DailyStats{
				UserID: userID,
				Date:   d,
			})
		}
	}
	return result
}

// truncateToDay 截断时间到当天 00:00:00 UTC
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (s *statsService) GetProjectStats(ctx context.Context, projectID, requesterID uint64, days int) (*ProjectStatsResponse, error) {
	// 仅项目 owner 可查看统计数据
	project, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerID != requesterID {
		return nil, apperrors.New(apperrors.CodePermissionDenied, "仅项目负责人可查看统计数据")
	}

	if days <= 0 || days > 90 {
		days = 30
	}
	to := time.Now()
	from := to.AddDate(0, 0, -days)

	daily, err := s.statsRepo.GetProjectDailyStats(ctx, projectID, from, to)
	if err != nil {
		return nil, err
	}

	// 补零填充，保证日期连续
	daily = fillProjectDailyStats(projectID, daily, from, to)

	// 汇总统计（全时）
	summary, err := s.statsRepo.GetProjectSummaryStats(ctx, projectID)
	if err != nil {
		// 汇总统计失败不阻断日历数据返回，降级为空对象
		summary = &repository.ProjectSummaryStats{}
	}

	period := fmt.Sprintf("%dd", days)

	return &ProjectStatsResponse{
		ProjectID: projectID,
		Summary:   summary,
		Daily:     daily,
		Period:    period,
	}, nil
}

// RunDailyAggregation 遍历所有活跃用户和活跃项目执行每日统计聚合。
// 活跃用户范围：当天有互动（新粉丝/被评论/被点赞）或前一天有历史统计记录。
// 分批处理：每批 aggregateBatchSize 条，批次间主动让出调度，防止大用户量时单次操作超时/OOM。
func (s *statsService) RunDailyAggregation(ctx context.Context, date time.Time) error {
	const aggregateBatchSize = 200

	// ————— 用户维度聚合 —————
	userIDs, err := s.statsRepo.ListActiveUserIDs(ctx, date)
	if err != nil {
		return fmt.Errorf("stats: list active users failed: %w", err)
	}

	var firstErr error
	for i := 0; i < len(userIDs); i += aggregateBatchSize {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("stats: aggregation cancelled: %w", err)
		}
		end := i + aggregateBatchSize
		if end > len(userIDs) {
			end = len(userIDs)
		}
		for _, uid := range userIDs[i:end] {
			us, err := s.statsRepo.AggregateUserDailyStats(ctx, uid, date)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("aggregate uid=%d: %w", uid, err)
				}
				continue
			}
			if err := s.statsRepo.UpsertDailyStats(ctx, us); err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("upsert uid=%d: %w", uid, err)
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	// ————— 项目维度聚合 —————
	projectIDs, err := s.statsRepo.ListActiveProjectIDs(ctx, date)
	if err != nil {
		if firstErr == nil {
			firstErr = fmt.Errorf("stats: list active projects failed: %w", err)
		}
		return firstErr
	}
	for i := 0; i < len(projectIDs); i += aggregateBatchSize {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("stats: aggregation cancelled: %w", err)
		}
		end := i + aggregateBatchSize
		if end > len(projectIDs) {
			end = len(projectIDs)
		}
		for _, pid := range projectIDs[i:end] {
			ps, err := s.statsRepo.AggregateProjectDailyStats(ctx, pid, date)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("aggregate project pid=%d: %w", pid, err)
				}
				continue
			}
			if err := s.statsRepo.UpsertProjectStats(ctx, ps); err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("upsert project pid=%d: %w", pid, err)
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 部分失败不阻断，返回第一个错误供日志记录
	return firstErr
}

// fillProjectDailyStats 对 [from, to] 区间补零，缺失日期补入空 ProjectStats
func fillProjectDailyStats(projectID uint64, existing []*model.ProjectStats, from, to time.Time) []*model.ProjectStats {
	byDate := make(map[string]*model.ProjectStats, len(existing))
	for _, s := range existing {
		byDate[s.Date.Format("2006-01-02")] = s
	}
	result := make([]*model.ProjectStats, 0, int(to.Sub(from).Hours()/24)+1)
	for d := truncateToDay(from); !d.After(truncateToDay(to)); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if rec, ok := byDate[key]; ok {
			result = append(result, rec)
		} else {
			result = append(result, &model.ProjectStats{
				ProjectID: projectID,
				Date:      d,
			})
		}
	}
	return result
}
