package authorization

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"errors"
	"math/big"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent/signingkey"
)

type Key struct {
	ID      string
	Private *rsa.PrivateKey
}

func InitializeKey(ctx context.Context, db *persistence.Database, box *cryptobox.Box) error {
	tx, err := db.SQL.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(603123010)"); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM "authorization".signing_keys WHERE active)`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return errors.New("active signing key already exists; initialization does not rotate keys")
	}
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO "authorization".signing_keys (id, algorithm, private_key, public_key, active) VALUES ($1, 'RS256', $2, $3, true)`, id, encrypted, public); err != nil {
		return err
	}
	return tx.Commit()
}

func LoadActiveKey(ctx context.Context, db *persistence.Database, box *cryptobox.Box) (*Key, error) {
	row, err := db.Ent.SigningKey.Query().Where(signingkey.Active(true)).Only(ctx)
	if err != nil {
		return nil, errors.New("signing key is unavailable; run the keys initialization command after migrations")
	}
	private, err := box.Open(row.PrivateKey, "signing-key:"+row.ID)
	if err != nil {
		return nil, errors.New("signing key could not be decrypted")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(private)
	if err != nil {
		return nil, errors.New("signing key is invalid")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok || row.Algorithm != "RS256" || key.N.BitLen() < 3072 {
		return nil, errors.New("signing key does not meet the configured algorithm and size")
	}
	return &Key{ID: row.ID, Private: key}, nil
}

func (k *Key) PublicJWK() map[string]string {
	return map[string]string{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": k.ID, "n": base64.RawURLEncoding.EncodeToString(k.Private.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.Private.E)).Bytes())}
}
