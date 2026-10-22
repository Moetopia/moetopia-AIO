package authorization

import (
	"context"
	"errors"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/signingkey"
)

func ValidateRestoredKeys(ctx context.Context, db *ent.Client, box *cryptobox.Box) error {
	db = persistence.Client(ctx, db)
	rows, err := db.SigningKey.Query().Where(signingkey.PrivateKeyNEQ([]byte{})).ForShare().All(ctx)
	if err != nil {
		return err
	}
	active := 0
	for _, row := range rows {
		if _, err := decryptKey(box, row); err != nil {
			return err
		}
		if row.Active {
			active++
		}
	}
	if active != 1 {
		return errors.New("exactly one decryptable active signing key required")
	}
	return nil
}

func DiscardRestoredCredentials(ctx context.Context, db *ent.Client) error {
	db = persistence.Client(ctx, db)
	if _, err := db.ProtocolSession.Delete().Exec(ctx); err != nil {
		return err
	}
	if _, err := db.AuthorizationGrant.Delete().Exec(ctx); err != nil {
		return err
	}
	if _, err := db.AuthorizationInteraction.Delete().Exec(ctx); err != nil {
		return err
	}
	_, err := db.LogoutInteraction.Delete().Exec(ctx)
	return err
}
