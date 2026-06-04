package service

import (
	"context"

	"github.com/gamero/gamero/internal/model"
	"github.com/gamero/gamero/internal/repository"
	"github.com/gamero/gamero/pkg/logger"
	"go.uber.org/zap"
)

// NotificationClient is a shared helper for sending notifications.
// Services embed this instead of duplicating notifSend/notifSendModel methods.
type NotificationClient struct {
	Svc      NotificationService
	prefRepo repository.NotificationPreferenceRepository
}

// SetNotificationService injects the notification service dependency.
func (nc *NotificationClient) SetNotificationService(svc NotificationService) {
	nc.Svc = svc
}

// SetNotificationPreferenceRepository injects the notification preference repository dependency.
func (nc *NotificationClient) SetNotificationPreferenceRepository(repo repository.NotificationPreferenceRepository) {
	nc.prefRepo = repo
}

// preferenceAllowed checks whether the user has enabled the given notification type.
// Returns true if no preference is set (default to allowed).
func (nc *NotificationClient) preferenceAllowed(ctx context.Context, userID uint64, notifType string) bool {
	if nc.prefRepo == nil {
		return true
	}
	pref, err := nc.prefRepo.GetPreference(ctx, userID, notifType)
	if err != nil {
		logger.Warn("failed to check notification preference", zap.Uint64("user_id", userID), zap.String("type", notifType), zap.Error(err))
		return true
	}
	if pref == nil {
		return true
	}
	return pref.Enabled
}

// Send sends a notification via the injected NotificationService.
// Skips sending if the recipient has disabled this notification type.
func (nc *NotificationClient) Send(ctx context.Context, req *SendNotificationReq) {
	if nc.Svc == nil {
		return
	}
	if !nc.preferenceAllowed(ctx, req.UserID, string(req.Type)) {
		return
	}
	if err := nc.Svc.Send(ctx, req); err != nil {
		logger.Warn("failed to send notification", zap.Error(err))
	}
}

// SendModel sends a pre-built notification model via the injected NotificationService.
// Skips sending if the recipient has disabled this notification type.
func (nc *NotificationClient) SendModel(ctx context.Context, n *model.Notification) {
	if nc.Svc == nil {
		return
	}
	if !nc.preferenceAllowed(ctx, n.UserID, string(n.Type)) {
		return
	}
	if err := nc.Svc.SendModel(ctx, n); err != nil {
		logger.Warn("failed to send notification", zap.Error(err))
	}
}
