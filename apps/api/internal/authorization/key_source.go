package authorization

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"time"

	"github.com/ory/fosite"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/signingkey"
)

type keySource interface {
	Signing(context.Context) (*Key, error)
	Public(context.Context, string) (*rsa.PublicKey, error)
	List(context.Context) ([]map[string]string, error)
}

type fixedKey struct{ key *Key }

func (s fixedKey) Signing(context.Context) (*Key, error) { return s.key, nil }
func (s fixedKey) Public(_ context.Context, id string) (*rsa.PublicKey, error) {
	if id != s.key.ID {
		return nil, fosite.ErrNotFound
	}
	return &s.key.Private.PublicKey, nil
}
func (s fixedKey) List(context.Context) ([]map[string]string, error) {
	return []map[string]string{s.key.PublicJWK()}, nil
}

type persistentKeys struct{ store *Store }

func (s persistentKeys) Signing(ctx context.Context) (*Key, error) {
	row, err := s.store.database(ctx).SigningKey.Query().Where(signingkey.Active(true)).ForShare().Only(ctx)
	if err != nil {
		return nil, errors.New("active signing key unavailable")
	}
	return decryptKey(s.store.box, row)
}
func (s persistentKeys) Public(ctx context.Context, id string) (*rsa.PublicKey, error) {
	row, err := s.store.database(ctx).SigningKey.Query().Where(signingkey.ID(id), signingkey.Or(signingkey.Active(true), signingkey.RetireAfterGT(time.Now()))).Select(signingkey.FieldID, signingkey.FieldAlgorithm, signingkey.FieldPublicKey).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		observeIntrospectionRead(ctx, err)
		return nil, err
	}
	key, err := parsePublic(row.Algorithm, row.PublicKey)
	if err != nil {
		observeIntrospectionRead(ctx, err)
	}
	return key, err
}
func (s persistentKeys) List(ctx context.Context) ([]map[string]string, error) {
	rows, err := s.store.database(ctx).SigningKey.Query().Where(signingkey.Or(signingkey.Active(true), signingkey.RetireAfterGT(time.Now()))).Select(signingkey.FieldID, signingkey.FieldAlgorithm, signingkey.FieldPublicKey).Order(ent.Asc(signingkey.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	keys := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		key, err := parsePublic(row.Algorithm, row.PublicKey)
		if err != nil {
			return nil, err
		}
		keys = append(keys, publicJWK(row.ID, key))
	}
	if len(keys) == 0 {
		return nil, errors.New("public signing keys unavailable")
	}
	return keys, nil
}
func parsePublic(algorithm string, data []byte) (*rsa.PublicKey, error) {
	parsed, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, errors.New("invalid signing public key")
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok || algorithm != "RS256" || key.N.BitLen() < 3072 || key.E < 3 {
		return nil, errors.New("invalid signing public key profile")
	}
	return key, nil
}

type signingKeyContext struct{}

func pinnedKey(ctx context.Context) (*Key, error) {
	key, ok := ctx.Value(signingKeyContext{}).(*Key)
	if !ok || key == nil {
		return nil, errors.New("signing key not pinned to transaction")
	}
	return key, nil
}
