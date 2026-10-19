package persistence

import (
	"context"
	"moetopia.local/aio/api/internal/persistence/ent"
)

type transactionClient = ent.Client
type Unit struct {
	*transactionClient
	tx *ent.Tx
}

func Begin(ctx context.Context, db *ent.Client) (*Unit, error) {
	if client := Client(ctx, db); client != db {
		return &Unit{transactionClient: client}, nil
	}
	tx, err := db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	return &Unit{transactionClient: tx.Client(), tx: tx}, nil
}
func (u *Unit) Client() *ent.Client { return u.transactionClient }
func (u *Unit) Commit() error {
	if u.tx == nil {
		return nil
	}
	return u.tx.Commit()
}
func (u *Unit) Rollback() error {
	if u.tx == nil {
		return nil
	}
	return u.tx.Rollback()
}
