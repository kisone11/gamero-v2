package model

// NotificationPreference 通知偏好表，记录用户对每种通知类型的开关状态。
type NotificationPreference struct {
	ID      uint64 `gorm:"primaryKey;autoIncrement"`
	UserID  uint64 `gorm:"not null;uniqueIndex:idx_user_type"`
	Type    string `gorm:"type:varchar(32);not null;uniqueIndex:idx_user_type"`
	Enabled bool   `gorm:"not null;default:true"`
}

// TableName 指定表名
func (NotificationPreference) TableName() string {
	return "notification_preferences"
}
