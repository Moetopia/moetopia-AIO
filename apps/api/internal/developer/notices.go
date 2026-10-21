package developer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
)

func (s *Service) notify(ctx context.Context, owner, id, event, reason string) error {
	if s.outbox == nil {
		return errors.New("application notifications are not configured")
	}
	email, err := s.users.VerifiedEmail(ctx, owner)
	if err != nil {
		return err
	}
	message := notifications.Message{
		To: email, Subject: "Moetopia application update / 应用状态更新",
		Text: fmt.Sprintf("Your Moetopia application %s has an update: %s.\n你的 Moetopia 应用状态已更新，请登录开发者中心查看详情。\n%s", id, event, reason),
	}
	return s.outbox.Enqueue(ctx, persistence.Client(ctx, s.db), message, time.Now().Add(24*time.Hour))
}
