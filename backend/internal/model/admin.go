// Package model 包含管理后台专用的数据模型。
package model

import "time"

// AdminAuditLog 管理员操作审计日志
// 记录管理员在后台执行的敏感操作，包括封禁用户、删帖、审批认证、处理举报、审核提现等。
type AdminAuditLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID    uint64    `gorm:"not null;index" json:"admin_id"`                     // 操作管理员 ID
	Action     string    `gorm:"type:varchar(64);not null;index" json:"action"`      // 操作类型
	TargetType string    `gorm:"type:varchar(32);not null" json:"target_type"`       // 目标类型: user/post/report/withdrawal/cert
	TargetID   uint64    `gorm:"not null" json:"target_id"`                          // 目标 ID
	Note       string    `gorm:"type:varchar(512);not null;default:''" json:"note"` // 备注
	CreatedAt  time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (AdminAuditLog) TableName() string { return "admin_audit_logs" }
