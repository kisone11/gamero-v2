// Package model 定义创作者数据统计相关的 GORM 数据模型。
// 本文件包含以下模型：
//   - DailyStats: 创作者每日统计快照（定时聚合写入）
//   - ProjectStats: 项目每日统计快照
package model

import "time"

// DailyStats 创作者每日统计快照（定时聚合写入）
// 对应数据库表 daily_stats
type DailyStats struct {
	ID     uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint64    `gorm:"not null;index:idx_daily_stats_user_date" json:"user_id"`
	Date   time.Time `gorm:"type:date;not null;index:idx_daily_stats_user_date" json:"date"`

	// 内容互动
	PostViews    int64 `gorm:"default:0" json:"post_views"`    // 帖子浏览量
	PostLikes    int64 `gorm:"default:0" json:"post_likes"`    // 帖子获赞
	PostComments int64 `gorm:"default:0" json:"post_comments"` // 帖子评论
	LogViews     int64 `gorm:"default:0" json:"log_views"`     // 日志浏览量
	LogLikes     int64 `gorm:"default:0" json:"log_likes"`     // 日志获赞

	// 粉丝
	NewFollowers int64 `gorm:"default:0" json:"new_followers"` // 新增粉丝数


	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (DailyStats) TableName() string { return "daily_stats" }

// ProjectStats 项目每日统计快照
// 对应数据库表 project_stats
type ProjectStats struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID uint64    `gorm:"not null;index:idx_project_stats_date" json:"project_id"`
	Date      time.Time `gorm:"type:date;not null;index:idx_project_stats_date" json:"date"`

	Views     int64 `gorm:"default:0" json:"views"`
	Likes     int64 `gorm:"default:0" json:"likes"`
	Followers int64 `gorm:"default:0" json:"followers"` // 当日累计关注数

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ProjectStats) TableName() string { return "project_stats" }
