// Package model 定义开发日志系统相关的 GORM 数据模型。
// 本文件包含以下模型：
//   - DevLog: 开发日志主表
//   - DevLogComment: 日志评论表
//   - DevLogLike: 日志点赞表
//   - DevLogCollect: 日志收藏表
//   - DevLogCommentLike: 评论点赞表
package model

import "time"

// DevLogType 日志类型枚举
type DevLogType string

const (
	DevLogTypeLog     DevLogType = "log"     // 普通日志
	DevLogTypeRelease DevLogType = "release" // 版本更新
)

// DevLogVisibility 日志可见范围枚举
type DevLogVisibility string

const (
	DevLogVisibilityPublic      DevLogVisibility = "public"       // 公开
	DevLogVisibilityMembersOnly DevLogVisibility = "members_only" // 仅项目成员可见
)

// DevLogStatus 日志状态枚举
type DevLogStatus string

const (
	DevLogStatusDraft     DevLogStatus = "draft"     // 草稿
	DevLogStatusPublished DevLogStatus = "published" // 已发布
)

// DevLog 开发日志主表
// 对应数据库表 dev_logs
type DevLog struct {
	ID           uint64           `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID    uint64           `gorm:"not null;index" json:"project_id"`                                      // 所属项目 ID，外键 projects.id
	AuthorID     uint64           `gorm:"not null;index" json:"author_id"`                                       // 作者 ID，外键 users.id
	Title        string           `gorm:"type:varchar(128);not null" json:"title"`                               // 日志标题，最多128字符
	Content      string           `gorm:"type:text" json:"content"`                                              // 日志正文（Markdown格式），最多50000字符
	LogType      DevLogType       `gorm:"type:varchar(16);not null;default:'log'" json:"log_type"`               // 日志类型：log / release
	Version      string           `gorm:"type:varchar(32);default:''" json:"version"`                            // 版本号，log_type=release 时必填
	Images       string           `gorm:"type:text;default:'[]'" json:"images"`                                  // 配图 对象存储 key 列表，JSON 存储 []string，最多6张
	Videos       string           `gorm:"type:text;default:'[]'" json:"videos"`                                  // 视频 对象存储 key 列表，JSON 存储 []string，最多3个
	DownloadURL   string           `gorm:"type:varchar(512);default:''" json:"download_url"`                      // 下载链接（release 类型使用）
	DownloadCount int64            `gorm:"column:download_count;not null;default:0" json:"download_count"`         // 下载次数
	Visibility    DevLogVisibility `gorm:"type:varchar(16);not null;default:'public'" json:"visibility"`          // 可见范围：public / members_only
	Status       DevLogStatus     `gorm:"type:varchar(16);not null;default:'draft'" json:"status"`               // 状态：draft / published
	ViewCount    int              `gorm:"not null;default:0" json:"view_count"`                                  // 浏览数
	LikeCount    int              `gorm:"not null;default:0" json:"like_count"`                                  // 点赞数（冗余字段）
	CommentCount int              `gorm:"not null;default:0" json:"comment_count"`                               // 评论数（冗余字段）
	CollectCount int              `gorm:"not null;default:0" json:"collect_count"`                               // 收藏数（冗余字段）
	CreatedAt    time.Time        `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time        `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt    *time.Time       `gorm:"index" json:"deleted_at,omitempty"` // 软删除时间戳
}

// TableName 指定表名
func (DevLog) TableName() string {
	return "dev_logs"
}

// DevLogComment 日志评论表
// 支持两级评论结构：一级评论（reply_to_id=0）和子评论（reply_to_id 指向父评论 ID）
// 对应数据库表 dev_log_comments
type DevLogComment struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	LogID     uint64     `gorm:"not null;index" json:"log_id"`                          // 所属日志 ID，外键 dev_logs.id
	AuthorID  uint64     `gorm:"not null;index" json:"author_id"`                       // 评论作者 ID，外键 users.id
	Content   string     `gorm:"type:varchar(1000);not null" json:"content"`            // 评论内容，最多1000字符
	ReplyToID uint64     `gorm:"not null;default:0" json:"reply_to_id"`                 // 回复的评论 ID，一级评论为 0
	LikeCount int        `gorm:"not null;default:0" json:"like_count"`                  // 评论点赞数（冗余字段）
	CreatedAt time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"` // 软删除时间戳
}

// TableName 指定表名
func (DevLogComment) TableName() string {
	return "dev_log_comments"
}

// DevLogLike 日志点赞表
// 唯一索引 (log_id, user_id) 防止重复点赞
// 对应数据库表 dev_log_likes
type DevLogLike struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	LogID     uint64    `gorm:"not null;uniqueIndex:idx_log_user_like" json:"log_id"`   // 日志 ID，外键 dev_logs.id
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_log_user_like" json:"user_id"`  // 用户 ID，外键 users.id
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (DevLogLike) TableName() string {
	return "dev_log_likes"
}

// DevLogCollect 日志收藏表
// 唯一索引 (log_id, user_id) 防止重复收藏
// 对应数据库表 dev_log_collects
type DevLogCollect struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	LogID     uint64    `gorm:"not null;uniqueIndex:idx_log_user_collect" json:"log_id"`  // 日志 ID，外键 dev_logs.id
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_log_user_collect" json:"user_id"` // 用户 ID，外键 users.id
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (DevLogCollect) TableName() string {
	return "dev_log_collects"
}

// DevLogCommentLike 评论点赞表
// 唯一索引 (comment_id, user_id) 防止重复点赞
// 对应数据库表 dev_log_comment_likes
type DevLogCommentLike struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CommentID uint64    `gorm:"not null;uniqueIndex:idx_comment_user_like" json:"comment_id"` // 评论 ID，外键 dev_log_comments.id
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_comment_user_like" json:"user_id"`    // 用户 ID，外键 users.id
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (DevLogCommentLike) TableName() string {
	return "dev_log_comment_likes"
}
