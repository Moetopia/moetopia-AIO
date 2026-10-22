package httpserver

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/authorization"
)

type ApplicationStatisticsOutput struct {
	Body authorization.ProtocolStatistics
}

func (a DeveloperEndpoints) registerStatistics(api huma.API) {
	huma.Get(api, "/api/v1/developer/applications/{id}/statistics", func(ctx context.Context, input *ApplicationGetInput) (*ApplicationStatisticsOutput, error) {
		user, err := a.actor(ctx, AccountWrite{Cookie: input.Cookie}, false)
		if err != nil {
			return nil, err
		}
		result, err := a.Service.Statistics(ctx, user.ID, input.ID)
		if err != nil {
			return nil, applicationError(err)
		}
		return &ApplicationStatisticsOutput{result}, nil
	})
}
