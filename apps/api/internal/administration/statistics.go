package administration

import (
	"context"
	"moetopia.local/aio/api/internal/notifications"
)

func (s *Service) NotificationStatistics(ctx context.Context) (notifications.QueueStatistics, error) {
	return s.outbox.Statistics(ctx)
}
