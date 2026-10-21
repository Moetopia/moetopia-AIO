package main

import (
	"context"
	"log/slog"
	"time"

	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/lifecycle"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/relyingparty"
	"moetopia.local/aio/api/internal/sessions"
)

func runWorkers(ctx context.Context, outbox *notifications.Outbox, accounts *identity.Service, web *sessions.Service, protocol *authorization.Provider, portal *relyingparty.Service, admin *administration.Service, accountLifecycle *lifecycle.Service) {
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
			if admin != nil {
				if err := admin.Cleanup(ctx); err != nil && ctx.Err() == nil {
					slog.Error("administrator security cleanup failed")
				}
			}
			for _, cleanup := range []func(context.Context) error{accountLifecycle.Cleanup, accounts.Cleanup, web.Cleanup, protocol.Cleanup, portal.Cleanup} {
				if err := cleanup(ctx); err != nil && ctx.Err() == nil {
					slog.Error("expired account state cleanup failed")
				}
			}
		}
	}
}
