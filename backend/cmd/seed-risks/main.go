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
	limit := flag.Int("limit", 8, "要填充风险的项目数量")
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
	if err := db.AutoMigrate(&model.ProjectRisk{}); err != nil {
		logger.Fatal("项目风险表迁移失败", zap.Error(err))
	}
	if err := seedRisks(db, *limit); err != nil {
		logger.Fatal("项目风险展示数据填充失败", zap.Error(err))
	}
	logger.Info("项目风险展示数据填充完成", zap.Int("project_limit", *limit))
}

func seedRisks(db *gorm.DB, limit int) error {
	if limit <= 0 {
		limit = 8
	}
	var projects []model.Project
	if err := db.Where("deleted_at IS NULL").Order("created_at DESC").Limit(limit).Find(&projects).Error; err != nil {
		return err
	}
	now := time.Now()
	for i := range projects {
		project := projects[i]
		dueSoon := now.AddDate(0, 0, 10+i)
		dueLater := now.AddDate(0, 0, 24+i)
		resolvedAt := now.AddDate(0, 0, -3)
		risks := []model.ProjectRisk{
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectRiskCategorySchedule, Level: model.ProjectRiskLevelHigh, Status: model.ProjectRiskStatusMitigating, Title: "核心版本排期压缩", Description: "关键玩法和内容制作同时进入收尾，测试窗口可能被压缩。", Mitigation: "冻结低优先级需求，优先保障可玩闭环和阻塞缺陷。", DueDate: &dueSoon},
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectRiskCategoryTech, Level: model.ProjectRiskLevelMedium, Status: model.ProjectRiskStatusOpen, Title: "性能波动影响试玩体验", Description: "复杂场景下帧率不稳定，可能影响演示和评测反馈。", Mitigation: "建立性能基准场景，逐项排查渲染、AI 和资源加载峰值。", DueDate: &dueLater},
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectRiskCategoryTeam, Level: model.ProjectRiskLevelLow, Status: model.ProjectRiskStatusResolved, Title: "成员交接信息不完整", Description: "部分任务说明缺少背景，影响接手效率。", Mitigation: "补充任务说明模板和资料库链接。", ResolvedAt: &resolvedAt},
		}
		for _, risk := range risks {
			if err := db.Where("project_id = ? AND title = ?", risk.ProjectID, risk.Title).Assign(risk).FirstOrCreate(&model.ProjectRisk{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
