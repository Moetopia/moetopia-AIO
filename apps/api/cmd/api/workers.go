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
	"moetopia.local/aio/api/internal/observability"
	"moetopia.local/aio/api/internal/relyingparty"
	"moetopia.local/aio/api/internal/sessions"
)

func runWorkers(ctx context.Context, outbox *notifications.Outbox, accounts *identity.Service, web *sessions.Service, protocol *authorization.Provider, portal *relyingparty.Service, admin *administration.Service, accountLifecycle *lifecycle.Service, metrics *observability.Registry) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastCleanup time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			started := time.Now()
			err := outbox.Deliver(ctx)
			metrics.ObserveJob("delivery", time.Since(started), err != nil)
			if err != nil && ctx.Err() == nil {
				slog.Error("notification delivery cycle failed")
			}
			if now.Sub(lastCleanup) < time.Minute {
				continue
			}
			lastCleanup = now
			if admin != nil {
				started := time.Now()
				err := admin.Cleanup(ctx)
				metrics.ObserveJob("administration", time.Since(started), err != nil)
				if err != nil && ctx.Err() == nil {
					slog.Error("administrator security cleanup failed")
				}
			}
			for _, task := range []struct {
				name string
				run  func(context.Context) error
			}{{"lifecycle", accountLifecycle.Cleanup}, {"identity", accounts.Cleanup}, {"sessions", web.Cleanup}, {"authorization", protocol.Cleanup}, {"relying_party", portal.Cleanup}} {
				started := time.Now()
				err := task.run(ctx)
				metrics.ObserveJob(task.name, time.Since(started), err != nil)
				if err != nil && ctx.Err() == nil {
					slog.Error("expired account state cleanup failed")
				}
			}
		}
	}
}
