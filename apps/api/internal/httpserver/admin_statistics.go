package httpserver

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"time"
)

type AdminStatisticsOutput struct {
	Body struct {
		Timezone      string                        `json:"timezone"`
		GeneratedAt   time.Time                     `json:"generated_at"`
		Identity      identity.IdentityStatistics   `json:"identity"`
		Applications  developer.Overview            `json:"applications"`
		Notifications notifications.QueueStatistics `json:"notifications"`
	}
}

func (a AdminEndpoints) registerStatistics(api huma.API) {
	huma.Get(api, "/api/v1/admin/statistics", func(ctx context.Context, input *AccountCookie) (*AdminStatisticsOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		if a.Developers == nil {
			return nil, huma.Error503ServiceUnavailable("Application statistics unavailable")
		}
		result := &AdminStatisticsOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			if result.Body.Identity, err = a.Accounts.Identity.Statistics(ctx); err != nil {
				return err
			}
			if result.Body.Applications, err = a.Developers.Overview(ctx); err != nil {
				return err
			}
			result.Body.Notifications, err = a.Service.NotificationStatistics(ctx)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		result.Body.Timezone = "UTC"
		result.Body.GeneratedAt = time.Now().UTC()
		return result, nil
	})
}
