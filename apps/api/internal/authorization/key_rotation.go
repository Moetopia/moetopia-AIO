package authorization

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/signingkey"
)

const signingKeyOverlap = 15 * time.Minute

func RotateKey(ctx context.Context, db *persistence.Database, box *cryptobox.Box) error {
	key, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return err
	}
	id := uuid.NewString()
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	encrypted, err := box.Seal(private, "signing-key:"+id)
	if err != nil {
		return err
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}
	unit, err := persistence.Begin(ctx, db.Ent)
	if err != nil {
		return err
	}
	defer unit.Rollback()
	current, err := unit.SigningKey.Query().Where(signingkey.Active(true)).ForUpdate().Only(ctx)
	if err != nil {
		return errors.New("rotation requires an existing active key; concurrent rotations must retry")
	}
	if _, err := decryptKey(box, current); err != nil {
		return err
	}
	if err := unit.SigningKey.UpdateOneID(current.ID).SetActive(false).SetRetireAfter(time.Now().Add(signingKeyOverlap)).Exec(ctx); err != nil {
		return err
	}
	if err := unit.SigningKey.Create().SetID(id).SetAlgorithm("RS256").SetPrivateKey(encrypted).SetPublicKey(public).SetActive(true).Exec(ctx); err != nil {
		return err
	}
	if err := audit.RecordTarget(ctx, unit.Client(), "system:keys-cli", "signing_key.rotated", id, "success"); err != nil {
		return err
	}
	return unit.Commit()
}
