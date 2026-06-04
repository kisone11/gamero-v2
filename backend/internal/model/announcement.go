package model

import "time"

// Announcement 站内公告/运营公告。
type Announcement struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string     `gorm:"type:varchar(120);not null" json:"title"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Level       string     `gorm:"type:varchar(16);not null;default:'info';index" json:"level"`
	IsActive    bool       `gorm:"not null;default:true;index" json:"is_active"`
	IsPinned    bool       `gorm:"not null;default:false;index" json:"is_pinned"`
	CreatedBy   uint64     `gorm:"not null;index" json:"created_by"`
	PublishedAt time.Time  `gorm:"not null;index" json:"published_at"`
	ExpiresAt   *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt   time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

func (Announcement) TableName() string { return "announcements" }
