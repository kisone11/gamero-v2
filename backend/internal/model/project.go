// Package model 包含项目系统的所有 GORM 数据模型。
// 本文件定义以下模型：
//   - Project: 游戏项目主表
//   - ProjectMember: 项目成员表
//   - ProjectTimeline: 项目时间轴事件表
package model

import (
	"encoding/json"
	"time"
)

// ===========================
// 枚举定义
// ===========================

// ProjectGenre 游戏类型枚举
type ProjectGenre string

const (
	ProjectGenreAction    ProjectGenre = "action"    // 动作
	ProjectGenreRPG       ProjectGenre = "rpg"       // RPG
	ProjectGenreStrategy  ProjectGenre = "strategy"  // 策略
	ProjectGenreSimulator ProjectGenre = "simulator" // 模拟
	ProjectGenrePuzzle    ProjectGenre = "puzzle"    // 解谜
	ProjectGenreHorror    ProjectGenre = "horror"    // 恐怖
	ProjectGenrePlatform  ProjectGenre = "platform"  // 平台
	ProjectGenreOther     ProjectGenre = "other"     // 其他
)

// ProjectStyleTag 游戏风格标签枚举
type ProjectStyleTag string

const (
	ProjectStyleTag2D        ProjectStyleTag = "2D"        // 2D
	ProjectStyleTag3D        ProjectStyleTag = "3D"        // 3D
	ProjectStyleTagPixel     ProjectStyleTag = "pixel"     // 像素
	ProjectStyleTagRealist   ProjectStyleTag = "realist"   // 写实
	ProjectStyleTagCartoon   ProjectStyleTag = "cartoon"   // 卡通
	ProjectStyleTagCyberpunk ProjectStyleTag = "cyberpunk" // 赛博朋克
	ProjectStyleTagFantasy   ProjectStyleTag = "fantasy"   // 奇幻
	ProjectStyleTagScifi     ProjectStyleTag = "scifi"     // 科幻
)

// ProjectStatus 项目状态枚举
type ProjectStatus string

const (
	ProjectStatusPreparing  ProjectStatus = "preparing"  // 立项中
	ProjectStatusDeveloping ProjectStatus = "developing" // 开发中
	ProjectStatusPlayable   ProjectStatus = "playable"   // 可玩Demo
	ProjectStatusLaunched   ProjectStatus = "launched"   // 已上线
	ProjectStatusPaused     ProjectStatus = "paused"     // 暂停
	ProjectStatusAbandoned  ProjectStatus = "abandoned"  // 放弃
)

// ProjectVisibility 项目可见性枚举
type ProjectVisibility string

const (
	ProjectVisibilityPublic  ProjectVisibility = "public"  // 公开（默认）
	ProjectVisibilityPrivate ProjectVisibility = "private" // 私密（仅成员可见）
)

// ProjectMemberRole 项目成员角色枚举
type ProjectMemberRole string

const (
	ProjectMemberRoleOwner          ProjectMemberRole = "owner"           // 负责人
	ProjectMemberRoleLeadProgrammer ProjectMemberRole = "lead_programmer" // 主程
	ProjectMemberRoleLeadArtist     ProjectMemberRole = "lead_artist"     // 主美
	ProjectMemberRoleLeadDesigner   ProjectMemberRole = "lead_designer"   // 主策
	ProjectMemberRoleSound          ProjectMemberRole = "sound"           // 音效
	ProjectMemberRoleTester         ProjectMemberRole = "tester"          // 测试
	ProjectMemberRoleMember         ProjectMemberRole = "member"          // 成员
)

// ProjectTaskStatus 项目任务状态枚举
type ProjectTaskStatus string

const (
	ProjectTaskStatusTodo  ProjectTaskStatus = "todo"
	ProjectTaskStatusDoing ProjectTaskStatus = "doing"
	ProjectTaskStatusDone  ProjectTaskStatus = "done"
)

// ProjectTaskPriority 项目任务优先级枚举
type ProjectTaskPriority string

const (
	ProjectTaskPriorityLow    ProjectTaskPriority = "low"
	ProjectTaskPriorityMedium ProjectTaskPriority = "medium"
	ProjectTaskPriorityHigh   ProjectTaskPriority = "high"
)

// TimelineEventType 时间轴事件类型枚举
type TimelineEventType string

const (
	TimelineEventCreated       TimelineEventType = "created"        // 项目创建
	TimelineEventStatusChanged TimelineEventType = "status_changed" // 状态变更
	TimelineEventDemoPublished TimelineEventType = "demo_published" // Demo发布
	TimelineEventLaunched      TimelineEventType = "launched"       // 项目上线
	TimelineEventEnded         TimelineEventType = "ended"          // 项目结束
	TimelineEventLogPublished  TimelineEventType = "log_published"  // 日志发布（预留）
)

// ValidProjectStatuses 有效的项目状态列表
var ValidProjectStatuses = map[ProjectStatus]bool{
	ProjectStatusPreparing:  true,
	ProjectStatusDeveloping: true,
	ProjectStatusPlayable:   true,
	ProjectStatusLaunched:   true,
	ProjectStatusPaused:     true,
	ProjectStatusAbandoned:  true,
}

// ValidProjectGenres 有效的游戏类型列表
var ValidProjectGenres = map[ProjectGenre]bool{
	ProjectGenreAction:    true,
	ProjectGenreRPG:       true,
	ProjectGenreStrategy:  true,
	ProjectGenreSimulator: true,
	ProjectGenrePuzzle:    true,
	ProjectGenreHorror:    true,
	ProjectGenrePlatform:  true,
	ProjectGenreOther:     true,
}

// ValidProjectStyleTags 有效的风格标签集合
var ValidProjectStyleTags = map[ProjectStyleTag]bool{
	ProjectStyleTag2D:        true,
	ProjectStyleTag3D:        true,
	ProjectStyleTagPixel:     true,
	ProjectStyleTagRealist:   true,
	ProjectStyleTagCartoon:   true,
	ProjectStyleTagCyberpunk: true,
	ProjectStyleTagFantasy:   true,
	ProjectStyleTagScifi:     true,
}

// GameEngine 游戏引擎枚举
type GameEngine string

const (
	GameEngineUnity     GameEngine = "Unity"
	GameEngineUnreal    GameEngine = "Unreal Engine"
	GameEngineGodot     GameEngine = "Godot"
	GameEngineCocos     GameEngine = "Cocos"
	GameEngineGameMaker GameEngine = "GameMaker"
	GameEngineRPGMaker  GameEngine = "RPG Maker"
	GameEngineCustom    GameEngine = "自研引擎"
	GameEngineOther     GameEngine = "其他"
)

// Platform 目标平台枚举
type Platform string

const (
	PlatformPC      Platform = "PC"
	PlatformMobile  Platform = "Mobile"
	PlatformWeb     Platform = "Web"
	PlatformConsole Platform = "Console"
)

// EngineOptions 引擎选项列表（供前端下拉选择）
var EngineOptions = []GameEngine{
	GameEngineUnity, GameEngineUnreal, GameEngineGodot,
	GameEngineCocos, GameEngineGameMaker, GameEngineRPGMaker,
	GameEngineCustom, GameEngineOther,
}

// StyleTagOptions 风格标签选项列表（供前端选择）
var StyleTagOptions = []ProjectStyleTag{
	ProjectStyleTag2D, ProjectStyleTag3D, ProjectStyleTagPixel,
	ProjectStyleTagRealist, ProjectStyleTagCartoon, ProjectStyleTagCyberpunk,
	ProjectStyleTagFantasy, ProjectStyleTagScifi,
}

// PlatformOptions 平台选项列表（供前端选择）
var PlatformOptions = []Platform{
	PlatformPC, PlatformMobile, PlatformWeb, PlatformConsole,
}

// ValidProjectMemberRoles 有效的成员角色列表（不包含 owner，owner 不可手动指定）
var ValidProjectMemberRoles = map[ProjectMemberRole]bool{
	ProjectMemberRoleLeadProgrammer: true,
	ProjectMemberRoleLeadArtist:     true,
	ProjectMemberRoleLeadDesigner:   true,
	ProjectMemberRoleSound:          true,
	ProjectMemberRoleTester:         true,
	ProjectMemberRoleMember:         true,
}

var ValidProjectTaskStatuses = map[ProjectTaskStatus]bool{
	ProjectTaskStatusTodo:  true,
	ProjectTaskStatusDoing: true,
	ProjectTaskStatusDone:  true,
}

var ValidProjectTaskPriorities = map[ProjectTaskPriority]bool{
	ProjectTaskPriorityLow:    true,
	ProjectTaskPriorityMedium: true,
	ProjectTaskPriorityHigh:   true,
}

// ===========================
// 数据模型定义
// ===========================

// Project 游戏项目主表
// 对应数据库表 projects
type Project struct {
	ID             uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerID        uint64            `gorm:"not null;index" json:"owner_id"`                               // 创建者/负责人 ID，外键 users.id
	Name           string            `gorm:"type:varchar(128);not null" json:"name"`                       // 项目名称
	Slug           string            `gorm:"type:varchar(128);uniqueIndex;not null" json:"slug"`           // URL友好标识，由 name 生成
	Description    string            `gorm:"type:text" json:"description"`                                 // 项目简介
	Genre          ProjectGenre      `gorm:"type:varchar(32)" json:"genre"`                                // 游戏类型
	StyleTags      string            `gorm:"type:text;default:'[]'" json:"style_tags"`                     // 风格标签，JSON 存储 []string
	Status         ProjectStatus     `gorm:"type:varchar(32);not null;default:'preparing'" json:"status"`  // 项目状态
	Visibility     ProjectVisibility `gorm:"type:varchar(16);not null;default:'public'" json:"visibility"` // 可见性：public/private
	CoverKey       string            `gorm:"type:varchar(512)" json:"cover_key,omitempty"`                 // 封面图 对象存储 key
	ScreenshotKeys string            `gorm:"type:text;default:'[]'" json:"screenshot_keys"`                // 截图 对象存储 key 列表，JSON 存储
	VideoKeys      string            `gorm:"type:text;default:'[]'" json:"video_keys"`                     // 视频 对象存储 key 列表，JSON 存储 []string，最多3个
	DemoURL        string            `gorm:"type:varchar(512)" json:"demo_url,omitempty"`                  // Demo下载链接
	StoreURL       string            `gorm:"type:varchar(512)" json:"store_url,omitempty"`                 // 商店页链接
	FollowerCount  int               `gorm:"not null;default:0" json:"follower_count"`                     // 关注数（冗余字段）
	IsBanned       bool              `gorm:"not null;default:false;index" json:"is_banned"`                // 是否被管理员下架/封禁（高频过滤字段，加索引）
	BanReason      string            `gorm:"type:varchar(256)" json:"ban_reason,omitempty"`                // 下架原因
	CreatedAt      time.Time         `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time         `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt      *time.Time        `gorm:"index" json:"deleted_at,omitempty"` // 软删除时间戳
}

// TableName 指定表名
func (Project) TableName() string {
	return "projects"
}

// ProjectMember 项目成员表
// 记录项目与用户的成员关系
// 对应数据库表 project_members
type ProjectMember struct {
	ID           uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID    uint64            `gorm:"not null;index:idx_proj_member,unique" json:"project_id"` // 项目 ID，外键 projects.id
	UserID       uint64            `gorm:"not null;index:idx_proj_member,unique" json:"user_id"`    // 用户 ID，外键 users.id
	Role         ProjectMemberRole `gorm:"type:varchar(32);not null" json:"role"`                   // 成员角色
	Contribution string            `gorm:"type:varchar(256)" json:"contribution,omitempty"`         // 贡献简介
	JoinedAt     time.Time         `gorm:"not null" json:"joined_at"`                               // 加入时间
	LeftAt       *time.Time        `gorm:"default:null" json:"left_at,omitempty"`                   // 离开时间（nil=仍在职）
	IsActive     bool              `gorm:"not null;default:true;index" json:"is_active"`            // 是否仍在职
	CreatedAt    time.Time         `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (ProjectMember) TableName() string {
	return "project_members"
}

// ProjectTimeline 项目时间轴事件表
// 自动记录项目生命周期中的所有重要事件
// 对应数据库表 project_timelines
type ProjectTimeline struct {
	ID        uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID uint64            `gorm:"not null;index" json:"project_id"`                 // 项目 ID，外键 projects.id
	Type      TimelineEventType `gorm:"type:varchar(32);not null" json:"event_type"`      // 事件类型
	Title     string            `gorm:"type:varchar(256);not null" json:"title"`          // 事件标题
	Content   string            `gorm:"type:text" json:"description"`                     // 事件详细描述
	Metadata  json.RawMessage   `gorm:"type:text;default:'{}'" json:"metadata,omitempty"` // 附加元数据，JSON 格式
	CreatedAt time.Time         `gorm:"not null;autoCreateTime" json:"created_at"`
}

// ProjectTask 项目任务表
type ProjectTask struct {
	ID          uint64              `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID   uint64              `gorm:"not null;index" json:"project_id"`
	CreatorID   uint64              `gorm:"not null;index" json:"creator_id"`
	AssigneeID  *uint64             `gorm:"index" json:"assignee_id,omitempty"`
	Title       string              `gorm:"type:varchar(120);not null" json:"title"`
	Description string              `gorm:"type:varchar(1000);default:null" json:"description,omitempty"`
	Status      ProjectTaskStatus   `gorm:"type:varchar(16);not null;default:'todo';index" json:"status"`
	Priority    ProjectTaskPriority `gorm:"type:varchar(16);not null;default:'medium'" json:"priority"`
	DueDate     *time.Time          `gorm:"type:date;default:null" json:"due_date,omitempty"`
	CreatedAt   time.Time           `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time           `gorm:"not null;autoUpdateTime" json:"updated_at"`

	Project  Project `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"-"`
	Creator  User    `gorm:"foreignKey:CreatorID" json:"-"`
	Assignee *User   `gorm:"foreignKey:AssigneeID" json:"-"`
}

func (ProjectTask) TableName() string { return "project_tasks" }

// TableName 指定表名
func (ProjectTimeline) TableName() string {
	return "project_timelines"
}

// ProjectCollect 项目收藏
type ProjectCollect struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID uint64    `gorm:"not null;uniqueIndex:idx_proj_user_collect" json:"project_id"` // 项目 ID
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_proj_user_collect" json:"user_id"`    // 用户 ID
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (ProjectCollect) TableName() string {
	return "project_collects"
}

// GameReview 玩家对游戏的评测（区别于 CollaborationReview 开发者互评）
type GameReview struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID    uint64     `gorm:"not null;uniqueIndex:idx_game_review" json:"project_id"` // 项目 ID
	UserID       uint64     `gorm:"not null;uniqueIndex:idx_game_review" json:"user_id"`    // 玩家 ID（每个用户对一个项目只能评测一次）
	Rating       int8       `gorm:"not null" json:"rating"`                                 // 评分：1-5 星
	Content      string     `gorm:"type:varchar(1000);not null;default:''" json:"content"`  // 评测内容，最多 1000 字
	LikeCount    int        `gorm:"not null;default:0" json:"like_count"`                   // 有用票数
	ReplyContent string     `gorm:"type:varchar(1000);default:''" json:"reply_content"`     // 项目作者回复内容
	RepliedAt    *time.Time `gorm:"default:null" json:"replied_at,omitempty"`               // 回复时间
	CreatedAt    time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (GameReview) TableName() string { return "game_reviews" }

// GameReviewLike 评测点赞（每用户只能点一次）
type GameReviewLike struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ReviewID uint64 `gorm:"not null;uniqueIndex:idx_review_like" json:"review_id"`
	UserID   uint64 `gorm:"not null;uniqueIndex:idx_review_like" json:"user_id"`
}

// TableName 指定表名
func (GameReviewLike) TableName() string { return "game_review_likes" }

// ProjectFollow 项目关注表
// 记录用户关注项目的关系，每条记录表示某用户关注了某项目
// 对应数据库表 project_follows
// 唯一约束 (project_id, user_id) 防止重复关注
type ProjectFollow struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID uint64    `gorm:"not null;uniqueIndex:idx_proj_follow_pair" json:"project_id"` // 项目 ID，外键 projects.id
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_proj_follow_pair" json:"user_id"`    // 用户 ID，外键 users.id
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (ProjectFollow) TableName() string { return "project_follows" }
