package authorization

import (
	"context"
	"errors"
	"github.com/ory/fosite/storage"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
)

type transactionKey struct{}
type transaction struct {
	tx       *ent.Tx
	nested   bool
	replayed *bool
}

func (s *Store) database(ctx context.Context) *ent.Client {
	if state, ok := ctx.Value(transactionKey{}).(transaction); ok {
		return state.tx.Client()
	}
	return s.client
}
func (s *Store) BeginTX(ctx context.Context) (context.Context, error) {
	if state, ok := ctx.Value(transactionKey{}).(transaction); ok {
		return context.WithValue(ctx, transactionKey{}, transaction{state.tx, true, state.replayed}), nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	ctx = persistence.WithClient(ctx, tx.Client())
	return context.WithValue(ctx, transactionKey{}, transaction{tx, false, new(bool)}), nil
}
func (s *Store) Commit(ctx context.Context) error {
	state, ok := ctx.Value(transactionKey{}).(transaction)
	if !ok {
		return errors.New("protocol transaction missing")
	}
	// Fosite's inner transaction must not commit before PKCE and OIDC handlers finish.
	if state.nested {
		return nil
	}
	return state.tx.Commit()
}
func (s *Store) Rollback(ctx context.Context) error {
	state, ok := ctx.Value(transactionKey{}).(transaction)
	if !ok {
		return errors.New("protocol transaction missing")
	}
	if state.nested {
		return nil
	}
	return state.tx.Rollback()
}

var _ storage.Transactional = (*Store)(nil)
