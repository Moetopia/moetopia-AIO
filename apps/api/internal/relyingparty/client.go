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
const AdminClientID = "moetopia-admin"
const AdminCallbackPath = "/api/v1/admin/callback"
const LogoutCallbackPath = "/api/v1/portal/logout/callback"
const AdminLogoutCallbackPath = "/api/v1/admin/logout/callback"

func clientSecret(encodedKey string) (string, error) {
	return profileSecret(encodedKey, false)
}
func profileSecret(encodedKey string, admin bool) (string, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 {
		return "", errors.New("invalid RP key configuration")
	}
	mac := hmac.New(sha256.New, key)
	purpose := "user-site"
	if admin {
		purpose = "admin"
	}
	_, _ = mac.Write([]byte("moetopia:first-party:" + purpose + ":client:v1"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func CheckClient(ctx context.Context, db *ent.Client, issuer, key string) error {
	return checkProfile(ctx, db, issuer, issuer, key, false)
}
func checkProfile(ctx context.Context, db *ent.Client, issuer, origin, key string, admin bool) error {
	id, callback := ClientID, CallbackPath
	if admin {
		id, callback = AdminClientID, AdminCallbackPath
	}
	row, err := db.OAuthClient.Get(ctx, id)
	if err != nil {
		return errors.New("user-site OIDC client unavailable; run the explicit first-party initialization task")
	}
	secret, err := profileSecret(key, admin)
	if err != nil {
		return err
	}
	if !row.Enabled || row.Public || row.OwnerID != "" || row.TestOnly || !slices.Equal(row.RedirectUris, []string{origin + callback}) || !slices.Equal(row.GrantTypes, []string{"authorization_code"}) || !slices.Equal(row.ResponseTypes, []string{"code"}) || !slices.Equal(row.Scopes, []string{"openid", "profile", "email"}) || !slices.Equal(row.Audiences, []string{issuer + "/api"}) || bcrypt.CompareHashAndPassword(row.SecretHash, []byte(secret)) != nil {
		return errors.New("user-site OIDC client configuration mismatch; refusing to overwrite existing configuration")
	}
	return nil
}

func InitializeClient(ctx context.Context, db *ent.Client, issuer, key string) error {
	return initializeProfile(ctx, db, issuer, issuer, key, false)
}
func InitializeAdminClient(ctx context.Context, db *ent.Client, issuer, origin, key string) error {
	if origin == "" || origin == issuer {
		return errors.New("independent admin origin required")
	}
	return initializeProfile(ctx, db, issuer, origin, key, true)
}
func initializeProfile(ctx context.Context, db *ent.Client, issuer, origin, key string, admin bool) error {
	id, callback, name := ClientID, CallbackPath, "Moetopia user site"
	if admin {
		id, callback, name = AdminClientID, AdminCallbackPath, "Moetopia administration"
	}
	_, err := db.OAuthClient.Get(ctx, id)
	if err == nil {
		return checkProfile(ctx, db, issuer, origin, key, admin)
	}
	if !ent.IsNotFound(err) {
		return err
	}
	secret, err := profileSecret(key, admin)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	logoutCallback := LogoutCallbackPath
	if admin {
		logoutCallback = AdminLogoutCallbackPath
	}
	_, err = db.OAuthClient.Create().SetID(id).SetName(name).SetPublic(false).SetEnabled(true).SetSecretHash(hash).SetRedirectUris([]string{origin + callback}).SetPostLogoutRedirectUris([]string{origin + logoutCallback}).SetGrantTypes([]string{"authorization_code"}).SetResponseTypes([]string{"code"}).SetScopes([]string{"openid", "profile", "email"}).SetAudiences([]string{issuer + "/api"}).Save(ctx)
	if ent.IsConstraintError(err) {
		return checkProfile(ctx, db, issuer, origin, key, admin)
	}
	return err
}
