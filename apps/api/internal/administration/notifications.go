package administration

import (
	"context"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"strings"
)

func (s *Service) FailedNotifications(ctx context.Context, after string) (notifications.FailedDeliveries, error) {
	return s.outbox.Failed(ctx, after)
}
func (s *Service) RetryNotification(ctx context.Context, actor, id string, attempts int, reason string) error {
	if !validReason(reason) {
		return ErrInvalid
	}
	if err := s.outbox.Retry(ctx, id, attempts); err != nil {
		return err
	}
	return audit.RecordReason(ctx, persistence.Client(ctx, s.db), actor, "notification.requeued", id, strings.TrimSpace(reason))
}
