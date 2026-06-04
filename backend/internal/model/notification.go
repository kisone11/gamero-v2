package model

import "time"

// ===========================
// 通知类型枚举
// ===========================

// NotificationType 通知类型枚举
type NotificationType string

const (
	// 组队与申请
	NotificationTypeRecruitmentApplied  NotificationType = "recruitment_applied"  // 有人申请加入项目
	NotificationTypeApplicationApproved NotificationType = "application_approved" // 申请被通过
	NotificationTypeApplicationRejected NotificationType = "application_rejected" // 申请被拒绝

	// 社交互动
	NotificationTypeMention              NotificationType = "mention"               // 在评论中被 @提及
	NotificationTypeNewFollower          NotificationType = "new_follower"          // 有新粉丝关注
	NotificationTypeProjectStatusChanged NotificationType = "project_status_changed" // 关注的项目状态变更
	NotificationTypePostLiked            NotificationType = "post_liked"            // 帖子被点赞
	NotificationTypePostCommented        NotificationType = "post_commented"        // 帖子被评论
	NotificationTypeLogLiked             NotificationType = "log_liked"             // 开发日志被点赞
	NotificationTypeLogCommented         NotificationType = "log_commented"         // 开发日志被评论

	// 系统消息
	NotificationTypeSystem NotificationType = "system" // 系统通知

	// 人才邀请
	NotificationTypeTalentInvited  NotificationType = "talent_invited"  // 收到项目邀请
	NotificationTypeInviteAccepted NotificationType = "invite_accepted" // 邀请被接受
	NotificationTypeInviteDeclined NotificationType = "invite_declined" // 邀请被拒绝

	// 项目管理（管理员操作）
	NotificationTypeProjectBanned   NotificationType = "project_banned"   // 项目被管理员下架
	NotificationTypeProjectUnbanned NotificationType = "project_unbanned" // 项目被管理员恢复上架

	// 举报处理
	NotificationTypeReportHandled NotificationType = "report_handled" // 举报处理结果通知

)

// ===========================
// Notification 数据模型
// ===========================

// Notification 通知表，对应数据库表 notifications。
type Notification struct {
	ID        uint64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64           `gorm:"not null;index" json:"user_id"`                        // 接收者用户 ID，外键 users.id
	SenderID  uint64           `gorm:"not null;default:0" json:"sender_id"`                  // 发送者用户 ID（0 表示系统消息）
	Type      NotificationType `gorm:"type:varchar(32);not null" json:"type"`                // 通知类型
	Title     string           `gorm:"type:varchar(128);not null" json:"title"`              // 通知标题
	Content   string           `gorm:"type:text;not null" json:"content"`                    // 通知内容
	IsRead    bool             `gorm:"not null;default:false" json:"is_read"`                // 是否已读
	Metadata  string           `gorm:"type:text;default:'{}'" json:"metadata"`               // 附加元数据，JSON 格式
	CreatedAt time.Time        `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (Notification) TableName() string {
	return "notifications"
}
