package authorization

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/signingkey"
)

const signingKeyOverlap = 15 * time.Minute

func RotateKey(ctx context.Context, db *persistence.Database, box *cryptobox.Box) error {
	return rotateKey(ctx, db, box, "", nil)
}

func EmergencyRotateKey(ctx context.Context, db *persistence.Database, box *cryptobox.Box, reason string, reset func(context.Context) error) error {
	if strings.TrimSpace(reason) == "" || !utf8.ValidString(reason) || utf8.RuneCountInString(reason) > 500 || reset == nil {
		return errors.New("maintenance reason and global revoker required")
	}
	return rotateKey(ctx, db, box, strings.TrimSpace(reason), reset)
}

func rotateKey(ctx context.Context, db *persistence.Database, box *cryptobox.Box, reason string, reset func(context.Context) error) error {
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
	ctx = persistence.WithClient(ctx, unit.Client())
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
	if reset != nil {
		if err := unit.SigningKey.Update().Where(signingkey.IDNEQ(id)).SetActive(false).SetRetireAfter(time.Now()).SetPrivateKey([]byte{}).Exec(ctx); err != nil {
			return err
		}
		if err := reset(ctx); err != nil {
			return err
		}
		if err := audit.RecordReason(ctx, unit.Client(), "system:keys-cli", "signing_key.emergency_rotated", id, reason); err != nil {
			return err
		}
		return unit.Commit()
	}
	if err := audit.RecordTarget(ctx, unit.Client(), "system:keys-cli", "signing_key.rotated", id, "success"); err != nil {
		return err
	}
	return unit.Commit()
}
