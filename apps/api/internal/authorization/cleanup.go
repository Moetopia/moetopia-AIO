package authorization

import (
	"context"
	"errors"
	"time"

	"github.com/ory/fosite"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationgrant"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationinteraction"
	"moetopia.local/aio/api/internal/persistence/ent/logoutinteraction"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
	"moetopia.local/aio/api/internal/persistence/ent/signingkey"
)

func (p *Provider) Cleanup(ctx context.Context) error {
	if _, err := p.store.client.LogoutInteraction.Delete().Where(logoutinteraction.ExpiresAtLTE(time.Now())).Exec(ctx); err != nil {
		return err
	}
	if _, err := p.store.client.SigningKey.Update().Where(signingkey.Active(false), signingkey.RetireAfterLTE(time.Now()), signingkey.PrivateKeyNEQ([]byte{})).SetPrivateKey([]byte{}).Save(ctx); err != nil {
		return err
	}
	if _, err := p.store.client.AuthorizationInteraction.Delete().Where(authorizationinteraction.ExpiresAtLT(time.Now())).Exec(ctx); err != nil {
		return err
	}
	ids, err := p.store.client.AuthorizationGrant.Query().Where(authorizationgrant.ExpiresAtLT(time.Now().Add(-11 * time.Minute))).Limit(100).IDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := p.cleanupGrant(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) cleanupGrant(ctx context.Context, id string) error {
	ctx, err := p.store.BeginTX(ctx)
	if err != nil {
		return err
	}
	defer p.store.Rollback(ctx)
	grant, err := p.store.grant(ctx, id)
	if errors.Is(err, fosite.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !grant.ExpiresAt.Before(time.Now().Add(-11 * time.Minute)) {
		return nil
	}
	// Retain spent credentials for the entire family lifetime so old replays remain detectable.
	if _, err := p.store.database(ctx).ProtocolSession.Delete().Where(protocolsession.RequestID(id)).Exec(ctx); err != nil {
		return err
	}
	if err := p.store.database(ctx).AuthorizationGrant.DeleteOneID(id).Exec(ctx); err != nil {
		return err
	}
	return p.store.Commit(ctx)
}
