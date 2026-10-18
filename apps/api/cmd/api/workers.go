package main

import (
	"context"
	"log/slog"
	"moetopia.local/aio/api/internal/authorization"
	"time"

	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/sessions"
)

func runWorkers(ctx context.Context, outbox *notifications.Outbox, accounts *identity.Service, web *sessions.Service, protocol *authorization.Provider) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastCleanup time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := outbox.Deliver(ctx); err != nil && ctx.Err() == nil {
				slog.Error("notification delivery cycle failed")
			}
			if now.Sub(lastCleanup) < time.Minute {
				continue
			}
			lastCleanup = now
			for _, cleanup := range []func(context.Context) error{accounts.Cleanup, web.Cleanup, protocol.Cleanup} {
				if err := cleanup(ctx); err != nil && ctx.Err() == nil {
					slog.Error("expired account state cleanup failed")
				}
			}
		}
	}
}
