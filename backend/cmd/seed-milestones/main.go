package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/pkg/database"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	limit := flag.Int("limit", 6, "要填充里程碑的项目数量")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}
	if err := logger.Init(cfg.Logger); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if _, err := database.Init(cfg.Database); err != nil {
		logger.Fatal("数据库连接失败", zap.Error(err))
	}
	defer database.Close()

	db := database.Get()
	if err := db.AutoMigrate(&model.ProjectMilestone{}); err != nil {
		logger.Fatal("里程碑表迁移失败", zap.Error(err))
	}
	if err := seedMilestones(db, *limit); err != nil {
		logger.Fatal("里程碑展示数据填充失败", zap.Error(err))
	}
	logger.Info("里程碑展示数据填充完成", zap.Int("project_limit", *limit))
}

func seedMilestones(db *gorm.DB, limit int) error {
	if limit <= 0 {
		limit = 6
	}

	var projects []model.Project
	if err := db.Where("deleted_at IS NULL").Order("created_at DESC").Limit(limit).Find(&projects).Error; err != nil {
		return err
	}

	now := time.Now()
	for i := range projects {
		project := projects[i]
		completedAt := now.AddDate(0, 0, -7)
		doneDue := now.AddDate(0, 0, -10)
		activeDue := now.AddDate(0, 0, 14+i)
		plannedDue := now.AddDate(0, 1, 7+i)

		items := []model.ProjectMilestone{
			{
				ProjectID: project.ID, CreatorID: project.OwnerID,
				Title:       "核心玩法原型完成",
				Description: "验证基础循环、输入手感和第一版关卡节奏。",
				Status:      model.ProjectMilestoneStatusDone,
				DueDate:     &doneDue, CompletedAt: &completedAt,
			},
			{
				ProjectID: project.ID, CreatorID: project.OwnerID,
				Title:       "内部试玩版本",
				Description: "整理任务列表、修复高优先级问题，并邀请成员进行小范围试玩。",
				Status:      model.ProjectMilestoneStatusActive,
				DueDate:     &activeDue,
			},
			{
				ProjectID: project.ID, CreatorID: project.OwnerID,
				Title:       "公开展示与招募更新",
				Description: "准备截图、演示视频和招募说明，用于项目页公开展示。",
				Status:      model.ProjectMilestoneStatusPlanned,
				DueDate:     &plannedDue,
			},
		}

		for _, item := range items {
			if err := db.Where("project_id = ? AND title = ?", item.ProjectID, item.Title).
				Assign(item).
				FirstOrCreate(&model.ProjectMilestone{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
