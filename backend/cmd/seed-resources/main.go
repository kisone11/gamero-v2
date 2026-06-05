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
	limit := flag.Int("limit", 8, "要填充资料的项目数量")
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
	if err := db.AutoMigrate(&model.ProjectResource{}); err != nil {
		logger.Fatal("项目资料库表迁移失败", zap.Error(err))
	}
	if err := seedResources(db, *limit); err != nil {
		logger.Fatal("项目资料展示数据填充失败", zap.Error(err))
	}
	logger.Info("项目资料展示数据填充完成", zap.Int("project_limit", *limit))
}

func seedResources(db *gorm.DB, limit int) error {
	if limit <= 0 {
		limit = 8
	}

	var projects []model.Project
	if err := db.Where("deleted_at IS NULL").Order("created_at DESC").Limit(limit).Find(&projects).Error; err != nil {
		return err
	}

	for i := range projects {
		project := projects[i]
		slug := project.Slug
		if slug == "" {
			slug = fmt.Sprintf("project-%d", project.ID)
		}

		resources := []model.ProjectResource{
			{
				ProjectID: project.ID, CreatorID: project.OwnerID,
				Category:    model.ProjectResourceCategoryDoc,
				Title:       "项目设计文档",
				URL:         fmt.Sprintf("https://docs.example.com/%s/game-design", slug),
				Description: "核心玩法、系统结构、版本目标和风险清单。",
				IsPinned:    true,
			},
			{
				ProjectID: project.ID, CreatorID: project.OwnerID,
				Category:    model.ProjectResourceCategoryCode,
				Title:       "主代码仓库",
				URL:         fmt.Sprintf("https://github.com/gamero-demo/%s", slug),
				Description: "项目主仓库、分支规范和提交说明。",
				IsPinned:    true,
			},
			{
				ProjectID: project.ID, CreatorID: project.OwnerID,
				Category:    model.ProjectResourceCategoryBuild,
				Title:       "最新试玩包",
				URL:         fmt.Sprintf("https://builds.example.com/%s/latest", slug),
				Description: "供成员测试和评审的最新可运行版本。",
			},
			{
				ProjectID: project.ID, CreatorID: project.OwnerID,
				Category:    model.ProjectResourceCategoryAsset,
				Title:       "美术素材库",
				URL:         fmt.Sprintf("https://assets.example.com/%s", slug),
				Description: "角色、场景、UI、音效素材目录。",
			},
			{
				ProjectID: project.ID, CreatorID: project.OwnerID,
				Category:    model.ProjectResourceCategoryReference,
				Title:       "竞品与参考资料",
				URL:         fmt.Sprintf("https://refs.example.com/%s", slug),
				Description: "玩法参考、视觉参考和市场调研链接。",
			},
		}

		for _, resource := range resources {
			if err := db.Where("project_id = ? AND title = ?", resource.ProjectID, resource.Title).
				Assign(resource).
				FirstOrCreate(&model.ProjectResource{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
