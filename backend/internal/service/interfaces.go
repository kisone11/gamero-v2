// Package service 通用接口声明。
// 此文件定义被多个 service 共享的接口，避免各 service 文件重复声明相同接口。
package service

import (
	"context"

	"github.com/gamero/gamero/internal/model"
)

// NotificationPusher WebSocket 通知推送接口（避免循环引用 ws 包）。
// ws.Hub 实现此接口；测试中使用 mockNotificationPusher。
type NotificationPusher interface {
	PushNotification(ctx context.Context, userID uint64, notification *model.Notification)
	PushUnreadCount(userID uint64, count int64)
}
