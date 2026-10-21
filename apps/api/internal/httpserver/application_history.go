package httpserver

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/developer"
)

type ApplicationHistoryInput struct {
	ApplicationGetInput
	Before int `query:"before" minimum:"0" maximum:"2147483647"`
}
type ApplicationHistoryOutput struct{ Body developer.VersionHistory }

func (a DeveloperEndpoints) registerHistory(api huma.API) {
	huma.Get(api, "/api/v1/developer/applications/{id}/versions", func(ctx context.Context, input *ApplicationHistoryInput) (*ApplicationHistoryOutput, error) {
		user, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.History(ctx, user.ID, input.ID, input.Before)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationHistoryOutput{result}, nil
	})
}

func (a AdminEndpoints) registerApplicationHistory(api huma.API) {
	huma.Get(api, "/api/v1/admin/applications/{id}/versions", func(ctx context.Context, input *ApplicationHistoryInput) (*ApplicationHistoryOutput, error) {
		user, web, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		if a.Developers == nil {
			return nil, huma.Error503ServiceUnavailable("Application service unavailable")
		}
		result := &ApplicationHistoryOutput{}
		err = a.Service.Run(ctx, user, web, func(ctx context.Context) error {
			var err error
			result.Body, err = a.Developers.AdminHistory(ctx, input.ID, input.Before)
			return err
		})
		if err != nil {
			return nil, adminError(err)
		}
		return result, nil
	})
}
