// Package model 包含组队系统的所有 GORM 数据模型。
// 本文件定义以下模型：
//   - Recruitment: 招募信息表
//   - RecruitmentApplication: 招募申请表
//   - CollaborationReview: 合作评价表
package model

import (
	"time"
)

// ===========================
// 枚举定义
// ===========================

// RecruitmentPosition 招募岗位枚举（与技能分类对应）
type RecruitmentPosition string

const (
	RecruitmentPositionProgram RecruitmentPosition = "program" // 程序
	RecruitmentPositionArt     RecruitmentPosition = "art"     // 美术
	RecruitmentPositionDesign  RecruitmentPosition = "design"  // 策划
	RecruitmentPositionSound   RecruitmentPosition = "sound"   // 音效
)

// CooperationType 合作方式枚举
type CooperationType string

const (
	CooperationTypeOnline  CooperationType = "online"  // 线上
	CooperationTypeOffline CooperationType = "offline" // 线下
	CooperationTypeHybrid  CooperationType = "hybrid"  // 混合
)

// ContactType 联系方式类型枚举
type ContactType string

const (
	ContactTypeWechat   ContactType = "wechat"   // 微信号
	ContactTypeQQ       ContactType = "qq"        // QQ号
	ContactTypePlatform ContactType = "platform"  // 平台私信（默认）
)

// RecruitmentStatus 招募状态枚举
type RecruitmentStatus string

const (
	RecruitmentStatusOpen    RecruitmentStatus = "open"    // 开放
	RecruitmentStatusClosed  RecruitmentStatus = "closed"  // 关闭
	RecruitmentStatusExpired RecruitmentStatus = "expired" // 已过期
)

// ApplicationStatus 申请状态枚举
type ApplicationStatus string

const (
	ApplicationStatusPending   ApplicationStatus = "pending"   // 待处理
	ApplicationStatusApproved  ApplicationStatus = "approved"  // 通过
	ApplicationStatusRejected  ApplicationStatus = "rejected"  // 拒绝
	ApplicationStatusWithdrawn ApplicationStatus = "withdrawn" // 已撤回
)

// ===========================
// 预设常量
// ===========================

// ValidRecruitmentPositions 有效的招募岗位集合
var ValidRecruitmentPositions = map[RecruitmentPosition]bool{
	RecruitmentPositionProgram: true,
	RecruitmentPositionArt:     true,
	RecruitmentPositionDesign:  true,
	RecruitmentPositionSound:   true,
}

// ValidCooperationTypes 有效的合作方式集合
var ValidCooperationTypes = map[CooperationType]bool{
	CooperationTypeOnline:  true,
	CooperationTypeOffline: true,
	CooperationTypeHybrid:  true,
}

// ValidContactTypes 有效的联系方式类型集合
var ValidContactTypes = map[ContactType]bool{
	ContactTypeWechat:   true,
	ContactTypeQQ:       true,
	ContactTypePlatform: true,
}

// PresetReviewTags 评价标签预设列表
var PresetReviewTags = []string{
	"代码质量高",
	"沟通顺畅",
	"认真负责",
	"创意丰富",
	"技术扎实",
	"按时交付",
	"团队协作",
	"美术出色",
	"策划细致",
}

// RecruitmentDefaultDays 招募有效期默认天数
const RecruitmentDefaultDays = 30

// RecruitmentMaxDays 招募有效期最大天数
const RecruitmentMaxDays = 90

// RecruitmentMaxHeadcount 招募最大需求人数
const RecruitmentMaxHeadcount = 10

// RecruitmentMinHeadcount 招募最小需求人数
const RecruitmentMinHeadcount = 1

// ===========================
// 数据模型定义
// ===========================

// Recruitment 招募信息表
// 记录项目的招募岗位信息
// 对应数据库表 recruitments
type Recruitment struct {
	ID              uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID       uint64            `gorm:"not null;index" json:"project_id"`                                       // 关联项目 ID，外键 projects.id
	OwnerID         uint64            `gorm:"not null;index" json:"owner_id"`                                         // 发布者用户 ID，外键 users.id
	Position        RecruitmentPosition `gorm:"type:varchar(32);not null" json:"position"`                            // 岗位分类：program/art/design/sound
	Headcount       int               `gorm:"not null;default:1" json:"headcount"`                                    // 需求人数，1-10
	Description     string            `gorm:"type:varchar(200);not null;default:''" json:"description"`               // 需求说明，最多200字
	CooperationType CooperationType   `gorm:"type:varchar(16);not null" json:"cooperation_type"`                      // 合作方式：online/offline/hybrid
	ContactType     ContactType       `gorm:"type:varchar(16);not null;default:'platform'" json:"contact_type"`       // 联系方式类型：wechat/qq/platform
	ContactInfo     string            `gorm:"type:varchar(128);default:null" json:"contact_info,omitempty"`           // 具体联系方式（platform类型时为空）
	ExpireAt        time.Time         `gorm:"not null" json:"expire_at"`                                              // 过期时间
	ExpiresAt       *time.Time        `gorm:"index" json:"expires_at,omitempty"`                                      // 自动过期时间，nil 表示永不过期
	Status          RecruitmentStatus `gorm:"type:varchar(16);not null;default:'open'" json:"status"`                 // 状态：open/closed/expired
	CreatedAt       time.Time         `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Recruitment) TableName() string {
	return "recruitments"
}

// IsExpired 判断招募是否已过期
func (r *Recruitment) IsExpired() bool {
	return time.Now().After(r.ExpireAt)
}

// IsOpen 判断招募是否处于开放状态
func (r *Recruitment) IsOpen() bool {
	return r.Status == RecruitmentStatusOpen && !r.IsExpired()
}

// RecruitmentApplication 招募申请表
// 记录用户对招募岗位的申请
// 对应数据库表 recruitment_applications
type RecruitmentApplication struct {
	ID             uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	RecruitmentID  uint64            `gorm:"not null;index" json:"recruitment_id"`                                  // 关联招募 ID，外键 recruitments.id
	ApplicantID    uint64            `gorm:"not null;index" json:"applicant_id"`                                    // 申请人用户 ID，外键 users.id
	Position       RecruitmentPosition `gorm:"type:varchar(32);not null" json:"position"`                           // 申请岗位
	Message        string            `gorm:"type:varchar(100);default:''" json:"message"`                           // 个人说明，最多100字
	Status         ApplicationStatus `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`             // 申请状态：pending/approved/rejected/withdrawn
	CreatedAt      time.Time         `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time         `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (RecruitmentApplication) TableName() string {
	return "recruitment_applications"
}

// CollaborationReview 合作评价表
// 记录项目成员之间的互评信息
// 对应数据库表 collaboration_reviews
// 唯一约束：(project_id, reviewer_id, reviewee_id) — 每对成员在同一项目只能互评一次
type CollaborationReview struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID   uint64    `gorm:"not null;uniqueIndex:idx_review_triple" json:"project_id"`                // 项目 ID，外键 projects.id
	ReviewerID  uint64    `gorm:"not null;uniqueIndex:idx_review_triple" json:"reviewer_id"`               // 评价人用户 ID，外键 users.id
	RevieweeID  uint64    `gorm:"not null;uniqueIndex:idx_review_triple" json:"reviewee_id"`               // 被评人用户 ID，外键 users.id
	Rating      int8      `gorm:"not null" json:"rating"`                                                  // 评分：1-5星
	Comment     string    `gorm:"type:varchar(500);not null;default:''" json:"comment"`                    // 文字评语，最多500字
	Tags        string    `gorm:"type:text;default:'[]'" json:"tags"`                                      // 评价标签，JSON 存储 []string
	Supplement  string    `gorm:"type:varchar(200);default:null" json:"supplement,omitempty"`              // 补充说明，最多200字，只能补充一次
	CreatedAt   time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (CollaborationReview) TableName() string {
	return "collaboration_reviews"
}

// HasSupplement 判断是否已有补充说明
func (r *CollaborationReview) HasSupplement() bool {
	return r.Supplement != ""
}

// ===========================
// 人才邀请
// ===========================

// InvitationStatus 人才邀请状态枚举
type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"   // 待回应
	InvitationStatusAccepted  InvitationStatus = "accepted"  // 已接受
	InvitationStatusDeclined  InvitationStatus = "declined"  // 已拒绝
	InvitationStatusWithdrawn InvitationStatus = "withdrawn" // 已撤回（项目主撤回）
	InvitationStatusExpired   InvitationStatus = "expired"   // 已过期
)

// TalentInvitation 人才邀请表
// 项目主通过人才库向特定开发者发送邀请
// 唯一约束 (project_id, talent_id, position)：同一项目同一岗位只能向同一人发送一条有效邀请
// 对应数据库表 talent_invitations
type TalentInvitation struct {
	ID          uint64              `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID   uint64              `gorm:"not null;index;uniqueIndex:idx_invite_unique" json:"project_id"`    // 关联项目 ID
	InviterID   uint64              `gorm:"not null;index" json:"inviter_id"`                                  // 邀请人（项目 owner）用户 ID
	TalentID    uint64              `gorm:"not null;index;uniqueIndex:idx_invite_unique" json:"talent_id"`     // 被邀请人用户 ID
	Position    RecruitmentPosition `gorm:"type:varchar(32);not null;uniqueIndex:idx_invite_unique" json:"position"` // 邀请岗位
	Message     string              `gorm:"type:varchar(200);not null;default:''" json:"message"`              // 邀请说明（最多200字）
	Status      InvitationStatus    `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`   // 状态
	ExpireAt    time.Time           `gorm:"not null" json:"expire_at"`                                         // 邀请过期时间（默认7天）
	CreatedAt   time.Time           `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time           `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (TalentInvitation) TableName() string {
	return "talent_invitations"
}

// IsExpired 判断邀请是否已过期
func (i *TalentInvitation) IsExpired() bool {
	return time.Now().After(i.ExpireAt)
}

// IsPending 判断邀请是否处于待回应状态
func (i *TalentInvitation) IsPending() bool {
	return i.Status == InvitationStatusPending && !i.IsExpired()
}
