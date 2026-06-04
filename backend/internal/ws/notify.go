package ws

import (
	"context"
	"encoding/json"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/pkg/metrics"
	"go.uber.org/zap"
)

// PushNotification 向指定用户推送通知（多实例广播）
func (h *Hub) PushNotification(ctx context.Context, userID uint64, notification *model.Notification) {
	msg := Message{
		Type:    "notification",
		Payload: notification,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("ws: marshal notification failed", zap.Error(err))
		return
	}
	metrics.WSMessageTotal.WithLabelValues(msg.Type).Inc()
	h.PublishToUser(ctx, userID, data)
}

// unreadPayload 未读数更新消息 payload
type unreadPayload struct {
	Count int64 `json:"count"`
}

// PushUnreadCount 推送未读通知数更新（多实例广播）
func (h *Hub) PushUnreadCount(userID uint64, count int64) {
	msg := Message{
		Type:    "unread_count",
		Payload: unreadPayload{Count: count},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("ws: marshal unread count failed", zap.Error(err))
		return
	}
	metrics.WSMessageTotal.WithLabelValues(msg.Type).Inc()
	h.PublishToUser(context.Background(), userID, data)
}
