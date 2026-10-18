package authorization

import (
	"context"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationinteraction"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
	"time"
)

func (p *Provider) Cleanup(ctx context.Context) error {
	if _, err := p.store.client.AuthorizationInteraction.Delete().Where(authorizationinteraction.ExpiresAtLT(time.Now())).Exec(ctx); err != nil {
		return err
	}
	// Retain spent codes beyond the last possible access-token expiry to detect replays.
	_, err := p.store.client.ProtocolSession.Delete().Where(protocolsession.ExpiresAtLT(time.Now().Add(-11 * time.Minute))).Exec(ctx)
	return err
}
