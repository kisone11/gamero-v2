// Package repository 提供数据库访问层，封装所有 GORM 操作。
package repository

import (
	"context"

	"github.com/gamero/gamero/internal/model"
	"gorm.io/gorm"
)

// NotificationPreferenceRepository 通知偏好仓储接口
type NotificationPreferenceRepository interface {
	// GetPreferences 批量加载指定用户的所有通知偏好
	GetPreferences(ctx context.Context, userID uint64) ([]model.NotificationPreference, error)

	// GetPreference 获取单条通知偏好（不存在时返回 nil, nil）
	GetPreference(ctx context.Context, userID uint64, notifType string) (*model.NotificationPreference, error)

	// SetPreference 创建或更新通知偏好（upsert）
	SetPreference(ctx context.Context, userID uint64, notifType string, enabled bool) error
}

// notificationPreferenceRepository 是 NotificationPreferenceRepository 接口的具体实现
type notificationPreferenceRepository struct {
	db *gorm.DB
}

// NewNotificationPreferenceRepository 创建 notificationPreferenceRepository 实例
func NewNotificationPreferenceRepository(db *gorm.DB) NotificationPreferenceRepository {
	return &notificationPreferenceRepository{db: db}
}

// GetPreferences 批量加载指定用户的所有通知偏好
func (r *notificationPreferenceRepository) GetPreferences(ctx context.Context, userID uint64) ([]model.NotificationPreference, error) {
	var prefs []model.NotificationPreference
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&prefs).Error
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		prefs = []model.NotificationPreference{}
	}
	return prefs, nil
}

// GetPreference 获取单条通知偏好（不存在时返回 nil, nil）
func (r *notificationPreferenceRepository) GetPreference(ctx context.Context, userID uint64, notifType string) (*model.NotificationPreference, error) {
	var pref model.NotificationPreference
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ?", userID, notifType).
		First(&pref).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pref, nil
}

// SetPreference 创建或更新通知偏好（upsert）
// 使用 ON CONFLICT 语法实现 upsert：当 (user_id, type) 唯一键冲突时更新 enabled 字段
func (r *notificationPreferenceRepository) SetPreference(ctx context.Context, userID uint64, notifType string, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&model.NotificationPreference{}).
		Where("user_id = ? AND type = ?", userID, notifType).
		Assign(model.NotificationPreference{Enabled: enabled}).
		FirstOrCreate(&model.NotificationPreference{
			UserID:  userID,
			Type:    notifType,
			Enabled: enabled,
		}).Error
}
