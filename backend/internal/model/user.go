// Package model 定义 Gamero 平台的数据库模型。
// 本文件包含用户系统相关的所有 GORM 模型:
//   - User: 用户主表
//   - UserSkill: 用户技能标签表
//   - Portfolio: 用户作品集表
//   - UserFollow: 用户关注关系表
package model

import (
	"time"
)

// SkillCategory 技能分类枚举
type SkillCategory string

const (
	SkillCategoryProgram SkillCategory = "program" // 程序
	SkillCategoryArt     SkillCategory = "art"     // 美术
	SkillCategoryDesign  SkillCategory = "design"  // 策划
	SkillCategorySound   SkillCategory = "sound"   // 音效
	SkillCategoryCustom  SkillCategory = "custom"  // 自定义
)

// SkillLevel 技能熟练度枚举
type SkillLevel string

const (
	SkillLevelBeginner     SkillLevel = "beginner"     // 初级
	SkillLevelIntermediate SkillLevel = "intermediate" // 中级
	SkillLevelAdvanced     SkillLevel = "advanced"     // 高级
)

// PortfolioType 作品集类型枚举
type PortfolioType string

const (
	PortfolioTypeGame PortfolioType = "game" // 游戏
	PortfolioTypeDemo PortfolioType = "demo" // Demo
	PortfolioTypeArt  PortfolioType = "art"  // 美术
	PortfolioTypeCode PortfolioType = "code" // 代码
)

// UserRole 用户角色枚举
type UserRole string

const (
	UserRoleUser       UserRole = "user"       // 普通用户
	UserRoleCreator    UserRole = "creator"    // 认证创作者
	UserRoleModerator  UserRole = "moderator"  // 版主
	UserRoleAdmin      UserRole = "admin"      // 管理员
	UserRoleSuperAdmin UserRole = "superadmin" // 超级管理员
)

// UserStatus 用户状态枚举
type UserStatus int

const (
	UserStatusActive   UserStatus = 1 // 正常
	UserStatusDisabled UserStatus = 0 // 禁用
)

// User 用户主表
// 对应数据库表 users
type User struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"type:varchar(32);uniqueIndex;not null" json:"username"`             // 唯一用户名(字母数字下划线)
	Nickname     string     `gorm:"type:varchar(32);not null;default:''" json:"nickname"`              // 昵称(2-32位)
	Phone        string     `gorm:"type:varchar(11);uniqueIndex;default:null" json:"phone,omitempty"`  // 手机号(11位),可为空
	Email        string     `gorm:"type:varchar(128);uniqueIndex;default:null" json:"email,omitempty"` // 邮箱,可为空
	PasswordHash string     `gorm:"type:varchar(256);default:null" json:"-"`                           // bcrypt 密码哈希,不输出到 JSON
	AvatarKey    string     `gorm:"type:varchar(512);default:null" json:"avatar_key,omitempty"`        // 对象存储路径,如 avatars/1/xxx.jpg
	Bio          string     `gorm:"type:varchar(300);default:null" json:"bio,omitempty"`               // 个人简介(最多300字)
	Role         UserRole   `gorm:"type:varchar(16);not null;default:'user'" json:"role"`              // 用户角色
	Status       UserStatus `gorm:"not null;default:1" json:"status"`                                  // 用户状态:1正常,0禁用
	BannedUntil  *time.Time `gorm:"default:null" json:"banned_until,omitempty"`                        // 封号到期时间,nil 表示永久封号或未封号
	IsBanned     bool       `gorm:"column:is_banned;not null;default:false" json:"is_banned"`          // 是否被封禁(冗余字段,与 status+banned_until 保持同步,方便前端直接判断)
	// ===== 人才合作偏好字段 =====
	IsAvailable    bool   `gorm:"column:is_available;not null;default:false;index" json:"is_available"` // 是否开放合作(加入人才库)
	CoopPreference string `gorm:"type:varchar(16);default:null" json:"coop_preference,omitempty"`       // 合作方式偏好:online/offline/hybrid
	Location       string `gorm:"type:varchar(64);default:null" json:"location,omitempty"`              // 所在地区(省市,如"广东 深圳")
	// ===== 统计冗余字段 =====
	FollowerCount  int       `gorm:"not null;default:0" json:"follower_count"`  // 粉丝数
	FollowingCount int       `gorm:"not null;default:0" json:"following_count"` // 关注数
	CreatedAt      time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	// 关联关系(预加载用,不直接写入)
	Skills    []UserSkill `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"skills,omitempty"`
	Portfolio []Portfolio `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"portfolio,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// UserSkill 用户技能标签表
// 每条记录代表用户拥有的一项技能
// 对应数据库表 user_skills
type UserSkill struct {
	ID          uint64        `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint64        `gorm:"not null;index" json:"user_id"`                               // 所属用户 ID
	Category    SkillCategory `gorm:"type:varchar(16);not null" json:"category"`                   // 技能分类:program/art/design/sound/custom
	Name        string        `gorm:"type:varchar(32);not null" json:"name"`                       // 技能名称,如 Unity、像素风、系统策划
	Level       SkillLevel    `gorm:"type:varchar(16);not null;default:'beginner'" json:"level"`   // 熟练度:beginner/intermediate/advanced
	Description string        `gorm:"type:varchar(200);default:null" json:"description,omitempty"` // 技能描述(选填,最多200字,如"用 Unity 做过 3 款上线手游,熟悉 ECS")
	IsCustom    bool          `gorm:"not null;default:false" json:"is_custom"`                     // 是否为自定义标签
	CreatedAt   time.Time     `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (UserSkill) TableName() string {
	return "user_skills"
}

// Portfolio 用户作品集表
// 定位:展示用户在平台外完成的历史作品(如 jam 参赛作、外包作品、商业项目等)。
// 平台内的项目通过 Project + ProjectMember 自动同步到个人主页,无需在此重复添加。
// 对应数据库表 portfolios
type Portfolio struct {
	ID          uint64        `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint64        `gorm:"not null;index" json:"user_id"`                                // 所属用户 ID
	Name        string        `gorm:"type:varchar(128);not null" json:"name"`                       // 作品名称
	Type        PortfolioType `gorm:"type:varchar(16);not null" json:"type"`                        // 作品类型:game/demo/art/code
	Link        string        `gorm:"type:varchar(512);default:null" json:"link,omitempty"`         // 作品链接(Steam/GitHub/Itch.io 等外部链接,平台外作品建议填写)
	ImageURL    string        `gorm:"type:varchar(512);default:null" json:"image_url,omitempty"`    // 封面图片 URL
	Description string        `gorm:"type:varchar(1000);default:null" json:"description,omitempty"` // 作品描述
	CreatedAt   time.Time     `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Portfolio) TableName() string {
	return "portfolios"
}

// UserFollow 用户关注关系表
// 记录用户之间的关注关系,follower_id 关注 followed_id
// 对应数据库表 user_follows
// 注意:此表只建模,关注 API 由后续关注系统模块实现
type UserFollow struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	FollowerID uint64    `gorm:"not null;index:idx_follow_pair,unique" json:"follower_id"` // 关注者 ID
	FollowedID uint64    `gorm:"not null;index:idx_follow_pair,unique" json:"followed_id"` // 被关注者 ID
	CreatedAt  time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (UserFollow) TableName() string {
	return "user_follows"
}

// UserBlock 用户拉黑关系表
// blocker_id 拉黑 blocked_id 后:
//   - blocked_id 的内容不在 blocker_id 的 Feed/发现页中出现
//   - blocked_id 无法查看/回复 blocker_id 的帖子/日志评论
//   - 双向屏蔽:若 A 拉黑 B,则 B 也看不到 A 的内容(由业务层控制)
type UserBlock struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	BlockerID uint64    `gorm:"not null;index:idx_block_pair,unique" json:"blocker_id"` // 拉黑发起者
	BlockedID uint64    `gorm:"not null;index:idx_block_pair,unique" json:"blocked_id"` // 被拉黑用户
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (UserBlock) TableName() string {
	return "user_blocks"
}

// PresetSkills 预设技能标签数据
// key: 分类,value: 技能名称列表
var PresetSkills = map[SkillCategory][]string{
	SkillCategoryProgram: {"Unity", "UE", "Godot", "原生开发", "Web前端", "后端开发"},
	SkillCategoryArt:     {"像素风", "3D建模", "UI设计", "概念设计", "角色设计", "场景设计"},
	SkillCategoryDesign:  {"系统策划", "数值策划", "文案写作", "关卡设计", "剧情设计"},
	SkillCategorySound:   {"音乐创作", "音效制作", "声音设计", "配音"},
}

// IsPresetSkill 判断给定分类和名称是否为预设技能
func IsPresetSkill(category SkillCategory, name string) bool {
	names, ok := PresetSkills[category]
	if !ok {
		return false
	}
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
