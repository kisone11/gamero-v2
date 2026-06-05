package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/pkg/database"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	limit := flag.Int("limit", 8, "要填充 QA 清单的项目数量")
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
	if err := db.AutoMigrate(&model.ProjectQACheckItem{}); err != nil {
		logger.Fatal("QA 验收表迁移失败", zap.Error(err))
	}
	if err := seedQAItems(db, *limit); err != nil {
		logger.Fatal("QA 验收展示数据填充失败", zap.Error(err))
	}
	logger.Info("QA 验收展示数据填充完成", zap.Int("project_limit", *limit))
}

func seedQAItems(db *gorm.DB, limit int) error {
	if limit <= 0 {
		limit = 8
	}

	var projects []model.Project
	if err := db.Where("deleted_at IS NULL").Order("created_at DESC").Limit(limit).Find(&projects).Error; err != nil {
		return err
	}

	for i := range projects {
		project := projects[i]
		items := []model.ProjectQACheckItem{
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectQACategoryGameplay, Status: model.ProjectQAStatusPassed, Title: "核心玩法闭环可完成", Description: "从开始到结算的最短流程可稳定完成。", EvidenceURL: "https://qa.example.com/gameplay-pass", IsRequired: true},
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectQACategoryPerformance, Status: model.ProjectQAStatusPending, Title: "目标设备性能压测", Description: "低配设备 30 分钟游玩无崩溃，帧率达到项目目标。", Note: "需要补充 Android 低端机数据。", IsRequired: true},
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectQACategoryBug, Status: model.ProjectQAStatusBlocked, Title: "P0/P1 Bug 清零", Description: "发布候选版本不得存在 P0/P1 未关闭问题。", Note: "当前仍有一个存档回滚问题待验证。", IsRequired: true},
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectQACategoryStore, Status: model.ProjectQAStatusPending, Title: "商店页素材与描述确认", Description: "截图、封面、简介、标签、年龄分级信息完成最终确认。", IsRequired: true},
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectQACategoryAudio, Status: model.ProjectQAStatusPassed, Title: "音频响度与循环检查", Description: "BGM、环境音、UI 音效响度一致且无明显爆音。", IsRequired: false},
		}

		for j := range items {
			item := items[j]
			var existing model.ProjectQACheckItem
			err := db.Where("project_id = ? AND title = ?", item.ProjectID, item.Title).First(&existing).Error
			if err == nil {
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}
			if err := db.Create(&item).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
