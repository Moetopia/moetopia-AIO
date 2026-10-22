package httpserver

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/notifications"
)

type AdminNotificationsOutput struct {
	Body notifications.FailedDeliveries
}
type AdminNotificationRetryInput struct {
	AccountWrite
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Attempts int    `json:"attempts" minimum:"5"`
		Reason   string `json:"reason" minLength:"1" maxLength:"500"`
	}
}

func (a AdminEndpoints) registerNotifications(api huma.API) {
	huma.Get(api, "/api/v1/admin/notifications/failed", func(ctx context.Context, input *AdminPrincipalsInput) (*AdminNotificationsOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result := &AdminNotificationsOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Service.FailedNotifications(ctx, input.After)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
	huma.Post(api, "/api/v1/admin/notifications/{id}/retry", func(ctx context.Context, input *AdminNotificationRetryInput) (*struct{}, error) {
		user, web, err := a.actor(ctx, input.AccountWrite, true)
		if err != nil {
			return nil, err
		}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			return a.Service.RetryNotification(ctx, user.ID, input.ID, input.Body.Attempts, input.Body.Reason)
		})
		if err != nil {
			return nil, adminError(err)
		}
		return nil, nil
	})
}
