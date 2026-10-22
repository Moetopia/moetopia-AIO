package developer

import (
	"context"
	"moetopia.local/aio/api/internal/authorization"
)

func (s *Service) Statistics(ctx context.Context, owner, id string) (authorization.ProtocolStatistics, error) {
	app, err := s.Detail(ctx, owner, id)
	if err != nil {
		return authorization.ProtocolStatistics{}, err
	}
	return s.protocol.Statistics(ctx, app.ClientID, owner)
}
