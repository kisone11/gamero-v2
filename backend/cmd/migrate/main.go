// cmd/migrate/main.go 数据库迁移程序。
// 使用 GORM AutoMigrate 将所有模型同步到数据库（创建表/添加列/添加索引）。
// 注意：AutoMigrate 不会删除列或缩短列类型，生产环境变更需谨慎评估。
//
// 使用方式：
//
//	go run cmd/migrate/main.go -config config/config.yaml
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

	// 初始化日志
	if err := logger.Init(cfg.Logger); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("开始数据库迁移...")

	// 初始化数据库连接
	if _, err := database.Init(cfg.Database); err != nil {
		logger.Fatal("数据库连接失败", zap.Error(err))
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Error("关闭数据库连接失败", zap.Error(err))
		}
	}()

	// 执行 AutoMigrate
	// 顺序：先迁移主表（users），再迁移依赖表（user_skills、portfolios、user_follows），最后迁移项目系统表和组队系统表
	if err := database.AutoMigrate(
		// 用户系统
		&model.User{},       // 用户主表
		&model.UserSkill{},  // 用户技能标签表
		&model.Portfolio{},  // 用户作品集表
		&model.UserFollow{}, // 用户关注关系表
		// 项目系统
		&model.Project{},          // 游戏项目主表
		&model.ProjectMember{},    // 项目成员表
		&model.ProjectTimeline{},  // 项目时间轴表
		&model.ProjectTask{},      // 项目任务表
		&model.ProjectMilestone{}, // 项目里程碑表
		&model.ProjectResource{},  // 项目资料库表
		&model.ProjectRisk{},      // 项目风险雷达表
		&model.ProjectCollect{},
		&model.GameReview{},
		&model.GameReviewLike{},
		// 组队系统
		&model.Recruitment{},            // 招募信息表
		&model.RecruitmentApplication{}, // 招募申请表
		&model.TalentInvitation{},       // 人才邀请表
		&model.CollaborationReview{},    // 合作评价表
		&model.Notification{},           // 通知表（预建，供后续模块使用）
		&model.NotificationPreference{}, // 通知偏好表
		// 开发日志系统
		&model.DevLog{},            // 开发日志主表
		&model.DevLogComment{},     // 日志评论表
		&model.DevLogLike{},        // 日志点赞表
		&model.DevLogCollect{},     // 日志收藏表
		&model.DevLogCommentLike{}, // 评论点赞表
		// 关注系统
		&model.ProjectFollow{}, // 项目关注表
		&model.TopicFollow{},   // 话题关注表
		// 社区系统
		&model.Topic{},           // 话题表
		&model.Post{},            // 帖子主表
		&model.PostComment{},     // 帖子评论表
		&model.PostLike{},        // 帖子点赞表
		&model.PostCollect{},     // 帖子收藏表
		&model.PostCommentLike{}, // 帖子评论点赞表
		&model.Report{},          // 内容举报表
		// 审计日志
		&model.AdminAuditLog{}, // 管理员操作审计日志表
		// 创作者数据统计
		&model.DailyStats{},   // 创作者每日统计快照表
		&model.ProjectStats{}, // 项目每日统计快照表
		// 敏感词表
		&model.SensitiveWord{}, // 敏感词持久化表
		// 运营公告
		&model.Announcement{}, // 站内公告表
	); err != nil {
		logger.Fatal("数据库迁移失败", zap.Error(err))
	}

	logger.Info("数据库迁移完成",
		zap.Strings("tables", []string{
			// 用户系统
			model.User{}.TableName(),
			model.UserSkill{}.TableName(),
			model.Portfolio{}.TableName(),
			model.UserFollow{}.TableName(),
			// 项目系统
			model.Project{}.TableName(),
			model.ProjectMember{}.TableName(),
			model.ProjectTimeline{}.TableName(),
			model.ProjectTask{}.TableName(),
			model.ProjectMilestone{}.TableName(),
			model.ProjectResource{}.TableName(),
			model.ProjectRisk{}.TableName(),
			// 组队系统
			model.Recruitment{}.TableName(),
			model.RecruitmentApplication{}.TableName(),
			model.TalentInvitation{}.TableName(),
			model.CollaborationReview{}.TableName(),
			model.Notification{}.TableName(),
			model.NotificationPreference{}.TableName(),
			// 开发日志系统
			model.DevLog{}.TableName(),
			model.DevLogComment{}.TableName(),
			model.DevLogLike{}.TableName(),
			model.DevLogCollect{}.TableName(),
			model.DevLogCommentLike{}.TableName(),
			// 关注系统
			model.ProjectFollow{}.TableName(),
			model.TopicFollow{}.TableName(),
			// 社区系统
			model.Topic{}.TableName(),
			model.Post{}.TableName(),
			model.PostComment{}.TableName(),
			model.PostLike{}.TableName(),
			model.PostCollect{}.TableName(),
			model.PostCommentLike{}.TableName(),
			model.Report{}.TableName(),
			// 审计日志
			model.AdminAuditLog{}.TableName(),
			// 创作者数据统计
			model.DailyStats{}.TableName(),
			model.ProjectStats{}.TableName(),
			// 实名认证
			// 敏感词
			model.SensitiveWord{}.TableName(),
			// 运营公告
			model.Announcement{}.TableName(),
		}),
	)

	// 获取原始 *gorm.DB 对象用于预建默认数据
	db := database.Get()

	// 预建默认话题（使用 FirstOrCreate 保证幂等，重复执行不会报错）
	logger.Info("开始预建默认话题...")
	defaultTopics := []model.Topic{
		{Name: "游戏开发", Description: "讨论游戏开发技术与经验", IsDefault: true, PostCount: 0},
		{Name: "求职招聘", Description: "寻找合作伙伴或工作机会", IsDefault: true, PostCount: 0},
		{Name: "作品展示", Description: "展示你的游戏作品和原创内容", IsDefault: true, PostCount: 0},
		{Name: "教程分享", Description: "分享学习资源和教程", IsDefault: true, PostCount: 0},
		{Name: "工具推荐", Description: "推荐好用的开发工具和资源", IsDefault: true, PostCount: 0},
		{Name: "游戏测评", Description: "分享游戏体验和评测", IsDefault: true, PostCount: 0},
		{Name: "行业动态", Description: "游戏行业最新动态与资讯", IsDefault: true, PostCount: 0},
		{Name: "闲聊水区", Description: "轻松闲聊，分享日常", IsDefault: true, PostCount: 0},
	}
	for _, t := range defaultTopics {
		topic := t // 避免循环变量捕获问题
		result := db.FirstOrCreate(&topic, model.Topic{Name: topic.Name})
		if result.Error != nil {
			logger.Error("预建默认话题失败",
				zap.String("name", topic.Name),
				zap.Error(result.Error),
			)
		}
	}
	logger.Info("默认话题预建完成", zap.Int("count", len(defaultTopics)))

}
