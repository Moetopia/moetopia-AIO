package relyingparty

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"slices"

	"golang.org/x/crypto/bcrypt"
	"moetopia.local/aio/api/internal/persistence/ent"
)

const ClientID = "moetopia-user-site"
const CallbackPath = "/api/v1/portal/callback"

func clientSecret(encodedKey string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 {
		return "", errors.New("invalid RP key configuration")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("moetopia:first-party:user-site:client:v1"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func CheckClient(ctx context.Context, db *ent.Client, issuer, key string) error {
	row, err := db.OAuthClient.Get(ctx, ClientID)
	if err != nil {
		return errors.New("user-site OIDC client unavailable; run the explicit first-party initialization task")
	}
	secret, err := clientSecret(key)
	if err != nil {
		return err
	}
	if !row.Enabled || row.Public || !slices.Equal(row.RedirectUris, []string{issuer + CallbackPath}) || !slices.Equal(row.GrantTypes, []string{"authorization_code"}) || !slices.Equal(row.ResponseTypes, []string{"code"}) || !slices.Equal(row.Scopes, []string{"openid", "profile", "email"}) || !slices.Equal(row.Audiences, []string{issuer + "/api"}) || bcrypt.CompareHashAndPassword(row.SecretHash, []byte(secret)) != nil {
		return errors.New("user-site OIDC client configuration mismatch; refusing to overwrite existing configuration")
	}
	return nil
}

func InitializeClient(ctx context.Context, db *ent.Client, issuer, key string) error {
	_, err := db.OAuthClient.Get(ctx, ClientID)
	if err == nil {
		return CheckClient(ctx, db, issuer, key)
	}
	if !ent.IsNotFound(err) {
		return err
	}
	secret, err := clientSecret(key)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.OAuthClient.Create().SetID(ClientID).SetName("Moetopia user site").SetPublic(false).SetEnabled(true).SetSecretHash(hash).SetRedirectUris([]string{issuer + CallbackPath}).SetGrantTypes([]string{"authorization_code"}).SetResponseTypes([]string{"code"}).SetScopes([]string{"openid", "profile", "email"}).SetAudiences([]string{issuer + "/api"}).Save(ctx)
	if ent.IsConstraintError(err) {
		return CheckClient(ctx, db, issuer, key)
	}
	return err
}
