package model

import "time"

// SensitiveWord 敏感词持久化表。
// 服务启动时从此表全量加载到内存 AC 自动机；
// 管理员通过后台 API 增删时同步写入此表。
// 对应数据库表 sensitive_words
type SensitiveWord struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Word      string    `gorm:"type:varchar(128);uniqueIndex;not null" json:"word"` // 小写存储，唯一索引
	CreatedBy uint64    `gorm:"not null;default:0" json:"created_by"`               // 创建者管理员 ID（0=系统内置）
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (SensitiveWord) TableName() string { return "sensitive_words" }
