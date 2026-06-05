package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gamero/gamero/config"
	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/pkg/database"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type demoUser struct {
	Username string
	Nickname string
	Email    string
	Bio      string
	Role     model.UserRole
	Location string
	Coop     string
	Skills   []model.UserSkill
}

type demoProject struct {
	Name        string
	Slug        string
	Owner       string
	Description string
	Genre       model.ProjectGenre
	StyleTags   []string
	Status      model.ProjectStatus
	Engine      string
}

func main() {
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
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

	if err := seed(database.Get()); err != nil {
		logger.Fatal("展示数据填充失败", zap.Error(err))
	}
	logger.Info("展示数据填充完成")
}

func seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		users, err := seedUsers(tx)
		if err != nil {
			return err
		}
		topics, err := seedTopics(tx)
		if err != nil {
			return err
		}
		projects, err := seedProjects(tx, users)
		if err != nil {
			return err
		}
		if err := seedProjectMembers(tx, users, projects); err != nil {
			return err
		}
		if err := seedProjectResources(tx, projects); err != nil {
			return err
		}
		if err := seedDevLogs(tx, users, projects); err != nil {
			return err
		}
		posts, err := seedPosts(tx, users, projects, topics)
		if err != nil {
			return err
		}
		if err := seedRecruitments(tx, users, projects); err != nil {
			return err
		}
		if err := seedSocial(tx, users, projects, posts, topics); err != nil {
			return err
		}
		if err := refreshCounters(tx); err != nil {
			return err
		}
		return nil
	})
}

func seedUsers(tx *gorm.DB) (map[string]*model.User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Gamero123"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	demos := []demoUser{
		{
			Username: "demo_pixel_alchemist", Nickname: "像素炼金师", Email: "demo-pixel@gamero.local",
			Bio: "独立游戏程序兼像素美术，喜欢做有物理反馈的小世界。", Role: model.UserRoleCreator, Location: "广东 深圳", Coop: "online",
			Skills: []model.UserSkill{{Category: model.SkillCategoryProgram, Name: "Godot", Level: model.SkillLevelAdvanced, Description: "熟悉 2D 工具链与编辑器扩展"}, {Category: model.SkillCategoryArt, Name: "像素风", Level: model.SkillLevelAdvanced, Description: "角色动画、瓦片地图、UI 图标"}},
		},
		{
			Username: "demo_system_designer", Nickname: "系统策划阿岚", Email: "demo-designer@gamero.local",
			Bio: "专注 Roguelite 成长曲线、经济系统和关卡节奏。", Role: model.UserRoleCreator, Location: "上海", Coop: "hybrid",
			Skills: []model.UserSkill{{Category: model.SkillCategoryDesign, Name: "系统策划", Level: model.SkillLevelAdvanced, Description: "数值框架、成长循环、掉落表"}, {Category: model.SkillCategoryDesign, Name: "关卡设计", Level: model.SkillLevelIntermediate, Description: "灰盒验证和难度曲线"}},
		},
		{
			Username: "demo_audio_mage", Nickname: "音频法师", Email: "demo-audio@gamero.local",
			Bio: "做互动音乐和怪物音效，熟悉 FMOD/Wwise。", Role: model.UserRoleUser, Location: "浙江 杭州", Coop: "online",
			Skills: []model.UserSkill{{Category: model.SkillCategorySound, Name: "音效制作", Level: model.SkillLevelAdvanced, Description: "武器、UI、环境氛围"}, {Category: model.SkillCategorySound, Name: "音乐创作", Level: model.SkillLevelIntermediate, Description: "动态分层音乐"}},
		},
		{
			Username: "demo_ue_engineer", Nickname: "UE 工程师 Neo", Email: "demo-ue@gamero.local",
			Bio: "Unreal 技术美术/玩法程序，关注性能和手感。", Role: model.UserRoleUser, Location: "北京", Coop: "offline",
			Skills: []model.UserSkill{{Category: model.SkillCategoryProgram, Name: "UE", Level: model.SkillLevelAdvanced, Description: "Gameplay Ability、Niagara、性能分析"}, {Category: model.SkillCategoryArt, Name: "3D建模", Level: model.SkillLevelIntermediate, Description: "场景模块和材质调优"}},
		},
		{
			Username: "demo_writer_mori", Nickname: "叙事作者森", Email: "demo-writer@gamero.local",
			Bio: "写世界观、支线和角色台词，偏爱温暖又有一点黑色幽默的故事。", Role: model.UserRoleUser, Location: "四川 成都", Coop: "online",
			Skills: []model.UserSkill{{Category: model.SkillCategoryDesign, Name: "文案写作", Level: model.SkillLevelAdvanced, Description: "角色弧光、任务文本、对白"}, {Category: model.SkillCategoryDesign, Name: "剧情设计", Level: model.SkillLevelAdvanced, Description: "多结局和碎片化叙事"}},
		},
	}

	result := make(map[string]*model.User, len(demos))
	for _, d := range demos {
		user := model.User{
			Username: d.Username, Nickname: d.Nickname, Email: d.Email, PasswordHash: string(passwordHash), Bio: d.Bio,
			Role: d.Role, Status: model.UserStatusActive, IsAvailable: true, CoopPreference: d.Coop, Location: d.Location,
		}
		if err := tx.Where("username = ?", user.Username).Assign(user).FirstOrCreate(&user).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("user_id = ?", user.ID).Delete(&model.UserSkill{}).Error; err != nil {
			return nil, err
		}
		for _, skill := range d.Skills {
			skill.UserID = user.ID
			skill.IsCustom = !model.IsPresetSkill(skill.Category, skill.Name)
			if err := tx.Create(&skill).Error; err != nil {
				return nil, err
			}
		}
		result[d.Username] = &user
	}
	return result, nil
}

func seedTopics(tx *gorm.DB) (map[string]*model.Topic, error) {
	defaults := []model.Topic{
		{Name: "游戏开发", Description: "讨论游戏开发技术与经验", IsDefault: true},
		{Name: "求职招聘", Description: "寻找合作伙伴或工作机会", IsDefault: true},
		{Name: "作品展示", Description: "展示你的游戏作品和原创内容", IsDefault: true},
		{Name: "教程分享", Description: "分享学习资源和教程", IsDefault: true},
		{Name: "工具推荐", Description: "推荐好用的开发工具和资源", IsDefault: true},
		{Name: "游戏测评", Description: "分享游戏体验和评测", IsDefault: true},
		{Name: "行业动态", Description: "游戏行业最新动态与资讯", IsDefault: true},
		{Name: "闲聊水区", Description: "轻松闲聊，分享日常", IsDefault: true},
	}
	topics := make(map[string]*model.Topic, len(defaults))
	for _, topic := range defaults {
		if err := tx.Where("name = ?", topic.Name).Assign(topic).FirstOrCreate(&topic).Error; err != nil {
			return nil, err
		}
		topics[topic.Name] = &topic
	}
	return topics, nil
}

func seedProjects(tx *gorm.DB, users map[string]*model.User) (map[string]*model.Project, error) {
	demos := []demoProject{
		{Name: "星尘工坊", Slug: "demo-stardust-workshop", Owner: "demo_pixel_alchemist", Genre: model.ProjectGenreSimulator, Status: model.ProjectStatusPlayable, Engine: "Godot", StyleTags: []string{"2D", "pixel", "scifi"}, Description: "一款在漂流小行星上经营工坊的模拟游戏。玩家采矿、加工零件、修复旧飞船，并逐步拼出星区失落文明的线索。"},
		{Name: "雾港夜行", Slug: "demo-fog-harbor", Owner: "demo_system_designer", Genre: model.ProjectGenrePuzzle, Status: model.ProjectStatusDeveloping, Engine: "Unity", StyleTags: []string{"2D", "cartoon", "fantasy"}, Description: "叙事解谜 + 轻潜行。主角在雾港寻找失踪的哥哥，通过港口居民的证词和潮汐机关还原一场旧案。"},
		{Name: "齿轮荒原", Slug: "demo-gear-wasteland", Owner: "demo_ue_engineer", Genre: model.ProjectGenreAction, Status: model.ProjectStatusPreparing, Engine: "Unreal Engine", StyleTags: []string{"3D", "realist", "cyberpunk"}, Description: "第三人称动作原型，核心是可拆装义体和高机动战斗。当前正在验证移动系统、近战反馈和敌人群体 AI。"},
		{Name: "月桥旅店", Slug: "demo-moonbridge-inn", Owner: "demo_writer_mori", Genre: model.ProjectGenreRPG, Status: model.ProjectStatusLaunched, Engine: "RPG Maker", StyleTags: []string{"2D", "pixel", "fantasy"}, Description: "温暖治愈向短篇 RPG。经营一间只在满月出现的旅店，接待来自不同时间线的旅人。"},
	}

	projects := make(map[string]*model.Project, len(demos))
	now := time.Now()
	for i, d := range demos {
		styleTags, _ := json.Marshal(d.StyleTags)
		created := now.AddDate(0, 0, -14+i*3)
		project := model.Project{
			OwnerID: users[d.Owner].ID, Name: d.Name, Slug: d.Slug, Description: d.Description, Genre: d.Genre,
			StyleTags: string(styleTags), Status: d.Status, Visibility: model.ProjectVisibilityPublic,
			DemoURL: fmt.Sprintf("https://example.com/%s/demo", d.Slug), StoreURL: fmt.Sprintf("https://example.com/%s", d.Slug),
			CreatedAt: created, UpdatedAt: created.Add(24 * time.Hour),
		}
		if err := tx.Where("slug = ?", project.Slug).Assign(project).FirstOrCreate(&project).Error; err != nil {
			return nil, err
		}
		projects[d.Slug] = &project
	}
	return projects, nil
}

func seedProjectMembers(tx *gorm.DB, users map[string]*model.User, projects map[string]*model.Project) error {
	type member struct {
		Project string
		User    string
		Role    model.ProjectMemberRole
		Note    string
	}
	members := []member{
		{"demo-stardust-workshop", "demo_pixel_alchemist", model.ProjectMemberRoleOwner, "项目主理人，程序与像素美术"},
		{"demo-stardust-workshop", "demo_audio_mage", model.ProjectMemberRoleSound, "环境音效和互动音乐"},
		{"demo-fog-harbor", "demo_system_designer", model.ProjectMemberRoleOwner, "系统策划与关卡节奏"},
		{"demo-fog-harbor", "demo_writer_mori", model.ProjectMemberRoleLeadDesigner, "剧情、角色和谜题文本"},
		{"demo-gear-wasteland", "demo_ue_engineer", model.ProjectMemberRoleOwner, "战斗原型和技术美术"},
		{"demo-gear-wasteland", "demo_audio_mage", model.ProjectMemberRoleSound, "战斗音效与混音"},
		{"demo-moonbridge-inn", "demo_writer_mori", model.ProjectMemberRoleOwner, "叙事、脚本和事件编辑"},
		{"demo-moonbridge-inn", "demo_pixel_alchemist", model.ProjectMemberRoleLeadArtist, "角色行走图和 UI 图标"},
	}
	for _, m := range members {
		row := model.ProjectMember{ProjectID: projects[m.Project].ID, UserID: users[m.User].ID, Role: m.Role, Contribution: m.Note, JoinedAt: time.Now().AddDate(0, -1, 0), IsActive: true}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "project_id"}, {Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"role", "contribution", "is_active", "updated_at"})}).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedProjectResources(tx *gorm.DB, projects map[string]*model.Project) error {
	for slug, project := range projects {
		resources := []model.ProjectResource{
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectResourceCategoryDoc, Title: "项目设计文档", URL: fmt.Sprintf("https://example.com/%s/design-doc", slug), Description: "核心玩法、系统结构和版本目标说明。", IsPinned: true},
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectResourceCategoryCode, Title: "代码仓库", URL: fmt.Sprintf("https://github.com/gamero-demo/%s", slug), Description: "项目主仓库和开发分支说明。"},
			{ProjectID: project.ID, CreatorID: project.OwnerID, Category: model.ProjectResourceCategoryBuild, Title: "最新试玩包", URL: fmt.Sprintf("https://example.com/%s/latest-build", slug), Description: "供成员测试的最新可运行版本。"},
		}
		for _, resource := range resources {
			if err := tx.Where("project_id = ? AND title = ?", resource.ProjectID, resource.Title).Assign(resource).FirstOrCreate(&model.ProjectResource{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedDevLogs(tx *gorm.DB, users map[string]*model.User, projects map[string]*model.Project) error {
	logs := []model.DevLog{
		{ProjectID: projects["demo-stardust-workshop"].ID, AuthorID: users["demo_pixel_alchemist"].ID, Title: "第一个可玩循环：采矿、熔炼、装配", Content: "本周完成了资源采集、熔炉加工和订单系统的闭环。下一步会把小行星生态做得更有层次。", LogType: model.DevLogTypeLog, Status: model.DevLogStatusPublished, Visibility: model.DevLogVisibilityPublic, ViewCount: 342, LikeCount: 28, CommentCount: 2, CollectCount: 7},
		{ProjectID: projects["demo-stardust-workshop"].ID, AuthorID: users["demo_audio_mage"].ID, Title: "为工坊机器做了三层动态音效", Content: "机器从空转到满载会逐步叠加低频、齿轮和蒸汽层，玩家不用看 UI 也能听出产线状态。", LogType: model.DevLogTypeLog, Status: model.DevLogStatusPublished, Visibility: model.DevLogVisibilityPublic, ViewCount: 188, LikeCount: 19, CommentCount: 1, CollectCount: 4},
		{ProjectID: projects["demo-fog-harbor"].ID, AuthorID: users["demo_system_designer"].ID, Title: "雾港第 2 章灰盒完成", Content: "潮汐机关现在支持三段水位，玩家需要在不同时间进入同一个街区，线索会发生变化。", LogType: model.DevLogTypeLog, Status: model.DevLogStatusPublished, Visibility: model.DevLogVisibilityPublic, ViewCount: 276, LikeCount: 22, CommentCount: 2, CollectCount: 6},
		{ProjectID: projects["demo-gear-wasteland"].ID, AuthorID: users["demo_ue_engineer"].ID, Title: "移动手感原型 v0.2", Content: "冲刺、滑铲、攀爬和空中转向已经能组合使用。后续要解决镜头和锁定目标的冲突。", LogType: model.DevLogTypeRelease, Version: "0.2.0", DownloadURL: "https://example.com/demo-gear-wasteland/v0.2", DownloadCount: 43, Status: model.DevLogStatusPublished, Visibility: model.DevLogVisibilityPublic, ViewCount: 421, LikeCount: 35, CommentCount: 3, CollectCount: 9},
		{ProjectID: projects["demo-moonbridge-inn"].ID, AuthorID: users["demo_writer_mori"].ID, Title: "上线后第一个小修补", Content: "修复旅人事件偶发重复触发的问题，并补充了两段结尾后的对话。", LogType: model.DevLogTypeRelease, Version: "1.0.1", DownloadURL: "https://example.com/demo-moonbridge-inn", DownloadCount: 126, Status: model.DevLogStatusPublished, Visibility: model.DevLogVisibilityPublic, ViewCount: 512, LikeCount: 44, CommentCount: 4, CollectCount: 13},
	}
	for i := range logs {
		log := logs[i]
		log.Images = "[]"
		log.Videos = "[]"
		if err := tx.Where("project_id = ? AND title = ?", log.ProjectID, log.Title).Assign(log).FirstOrCreate(&log).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedPosts(tx *gorm.DB, users map[string]*model.User, projects map[string]*model.Project, topics map[string]*model.Topic) (map[string]*model.Post, error) {
	topicIDs := func(names ...string) string {
		ids := make([]uint64, 0, len(names))
		for _, name := range names {
			ids = append(ids, topics[name].ID)
		}
		b, _ := json.Marshal(ids)
		return string(b)
	}
	posts := []model.Post{
		{AuthorID: users["demo_pixel_alchemist"].ID, Title: "Godot 做 2D 工坊游戏时，状态机应该拆多细？", Content: "目前机器、角色和订单都用了状态机，但担心后面状态爆炸。大家会把生产线逻辑拆到资源组件里吗？", TopicIDs: topicIDs("游戏开发", "工具推荐"), ProjectID: projects["demo-stardust-workshop"].ID, ViewCount: 386, LikeCount: 24, CommentCount: 2, HotScore: 96.5},
		{AuthorID: users["demo_system_designer"].ID, Title: "分享一个 Roguelite 掉落表的小工具", Content: "做了一个表格脚本，可以按权重、保底和章节进度动态生成掉落池。适合早期快速试错。", TopicIDs: topicIDs("教程分享", "工具推荐"), ViewCount: 291, LikeCount: 31, CommentCount: 2, HotScore: 103.1},
		{AuthorID: users["demo_writer_mori"].ID, Title: "月桥旅店上线了，欢迎试玩和吐槽文本节奏", Content: "这是一个 40 分钟左右的短篇 RPG，主题是离别和再见。想听听大家对支线文本密度的反馈。", TopicIDs: topicIDs("作品展示", "游戏测评"), ProjectID: projects["demo-moonbridge-inn"].ID, ViewCount: 642, LikeCount: 58, CommentCount: 3, HotScore: 155.2},
		{AuthorID: users["demo_audio_mage"].ID, Title: "招募项目音频前，最好先准备哪些参考？", Content: "建议至少准备玩法视频、情绪关键词、禁用参考和目标平台。否则音频很容易做成漂亮但不服务玩法的东西。", TopicIDs: topicIDs("求职招聘", "教程分享"), ViewCount: 223, LikeCount: 18, CommentCount: 1, HotScore: 72.4},
	}
	result := make(map[string]*model.Post, len(posts))
	for i := range posts {
		post := posts[i]
		post.Images = "[]"
		post.Videos = "[]"
		if err := tx.Where("title = ?", post.Title).Assign(post).FirstOrCreate(&post).Error; err != nil {
			return nil, err
		}
		result[post.Title] = &post
	}
	comments := []model.PostComment{
		{PostID: result["Godot 做 2D 工坊游戏时，状态机应该拆多细？"].ID, AuthorID: users["demo_ue_engineer"].ID, Content: "我会把状态机留给表现层，生产规则放到数据驱动系统里，调试会轻很多。"},
		{PostID: result["Godot 做 2D 工坊游戏时，状态机应该拆多细？"].ID, AuthorID: users["demo_system_designer"].ID, Content: "订单系统可以先抽象成目标和奖励，不要和机器强绑定。"},
		{PostID: result["月桥旅店上线了，欢迎试玩和吐槽文本节奏"].ID, AuthorID: users["demo_audio_mage"].ID, Content: "旅店夜晚 BGM 的留白很好，文本节奏后半段略快。"},
		{PostID: result["月桥旅店上线了，欢迎试玩和吐槽文本节奏"].ID, AuthorID: users["demo_pixel_alchemist"].ID, Content: "美术气质很统一，菜单交互可以再轻一点。"},
	}
	for i := range comments {
		comment := comments[i]
		if err := tx.Where("post_id = ? AND author_id = ? AND content = ?", comment.PostID, comment.AuthorID, comment.Content).FirstOrCreate(&comment).Error; err != nil {
			return nil, err
		}
	}
	return result, nil
}

func seedRecruitments(tx *gorm.DB, users map[string]*model.User, projects map[string]*model.Project) error {
	expire := time.Now().AddDate(0, 0, 30)
	recruitments := []model.Recruitment{
		{ProjectID: projects["demo-stardust-workshop"].ID, OwnerID: users["demo_pixel_alchemist"].ID, Position: model.RecruitmentPositionDesign, Headcount: 1, Description: "寻找系统策划一起打磨生产链和订单经济。", CooperationType: model.CooperationTypeOnline, ContactType: model.ContactTypePlatform, ExpireAt: expire, ExpiresAt: &expire, Status: model.RecruitmentStatusOpen},
		{ProjectID: projects["demo-fog-harbor"].ID, OwnerID: users["demo_system_designer"].ID, Position: model.RecruitmentPositionArt, Headcount: 2, Description: "需要 2D 场景美术，参与雾港街区、码头和室内资产。", CooperationType: model.CooperationTypeHybrid, ContactType: model.ContactTypePlatform, ExpireAt: expire, ExpiresAt: &expire, Status: model.RecruitmentStatusOpen},
		{ProjectID: projects["demo-gear-wasteland"].ID, OwnerID: users["demo_ue_engineer"].ID, Position: model.RecruitmentPositionProgram, Headcount: 1, Description: "招募玩法程序，协助敌人 AI、锁定系统和手柄适配。", CooperationType: model.CooperationTypeOffline, ContactType: model.ContactTypePlatform, ExpireAt: expire, ExpiresAt: &expire, Status: model.RecruitmentStatusOpen},
	}
	for i := range recruitments {
		rec := recruitments[i]
		if err := tx.Where("project_id = ? AND position = ?", rec.ProjectID, rec.Position).Assign(rec).FirstOrCreate(&rec).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedSocial(tx *gorm.DB, users map[string]*model.User, projects map[string]*model.Project, posts map[string]*model.Post, topics map[string]*model.Topic) error {
	for _, pair := range [][2]string{{"demo_system_designer", "demo_pixel_alchemist"}, {"demo_audio_mage", "demo_pixel_alchemist"}, {"demo_pixel_alchemist", "demo_writer_mori"}, {"demo_ue_engineer", "demo_audio_mage"}} {
		row := model.UserFollow{FollowerID: users[pair[0]].ID, FollowedID: users[pair[1]].ID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
	}
	for _, p := range projects {
		for _, u := range []string{"demo_system_designer", "demo_audio_mage", "demo_writer_mori"} {
			if users[u].ID == p.OwnerID {
				continue
			}
			row := model.ProjectFollow{ProjectID: p.ID, UserID: users[u].ID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
	}
	for _, topic := range topics {
		for _, u := range []string{"demo_pixel_alchemist", "demo_system_designer"} {
			row := model.TopicFollow{TopicID: topic.ID, UserID: users[u].ID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
	}
	reviews := []model.GameReview{
		{ProjectID: projects["demo-moonbridge-inn"].ID, UserID: users["demo_pixel_alchemist"].ID, Rating: 5, Content: "短小但完整，旅人故事很有记忆点。", LikeCount: 8},
		{ProjectID: projects["demo-stardust-workshop"].ID, UserID: users["demo_system_designer"].ID, Rating: 4, Content: "工坊反馈很舒服，期待更复杂的订单链。", LikeCount: 5},
		{ProjectID: projects["demo-gear-wasteland"].ID, UserID: users["demo_audio_mage"].ID, Rating: 4, Content: "移动很爽，打击音效还有很大空间。", LikeCount: 3},
	}
	for i := range reviews {
		review := reviews[i]
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "project_id"}, {Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"rating", "content", "like_count", "updated_at"})}).Create(&review).Error; err != nil {
			return err
		}
	}
	_ = posts
	return nil
}

func refreshCounters(tx *gorm.DB) error {
	queries := []string{
		`UPDATE users u SET follower_count = COALESCE(x.cnt, 0) FROM (SELECT followed_id, COUNT(*)::int cnt FROM user_follows GROUP BY followed_id) x WHERE u.id = x.followed_id`,
		`UPDATE users u SET following_count = COALESCE(x.cnt, 0) FROM (SELECT follower_id, COUNT(*)::int cnt FROM user_follows GROUP BY follower_id) x WHERE u.id = x.follower_id`,
		`UPDATE projects p SET follower_count = COALESCE(x.cnt, 0) FROM (SELECT project_id, COUNT(*)::int cnt FROM project_follows GROUP BY project_id) x WHERE p.id = x.project_id`,
		`UPDATE topics t SET follower_count = COALESCE(x.cnt, 0) FROM (SELECT topic_id, COUNT(*)::int cnt FROM topic_follows GROUP BY topic_id) x WHERE t.id = x.topic_id`,
		`UPDATE topics t SET post_count = COALESCE(x.cnt, 0) FROM (SELECT jsonb_array_elements_text(topic_ids::jsonb)::bigint topic_id, COUNT(*)::int cnt FROM posts WHERE deleted_at IS NULL GROUP BY topic_id) x WHERE t.id = x.topic_id`,
	}
	for _, query := range queries {
		if err := tx.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}
