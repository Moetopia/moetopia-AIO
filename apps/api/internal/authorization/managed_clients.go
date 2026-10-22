package authorization

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationgrant"
	"moetopia.local/aio/api/internal/persistence/ent/oauthclient"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
)

type ManagedClient struct {
	ID, OwnerID, Name         string
	Public, Enabled, TestOnly bool
	RedirectURIs, Scopes      []string
	PostLogoutRedirectURIs    []string
}

func NewClientSecret() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	return secret, hash, err
}
func (p *Provider) InstallManagedClient(ctx context.Context, spec ManagedClient, hash []byte) error {
	if spec.PostLogoutRedirectURIs == nil {
		spec.PostLogoutRedirectURIs = []string{}
	}
	if !strings.HasPrefix(spec.ID, "mt_app_") || spec.OwnerID == "" || spec.Name == "" || len(spec.RedirectURIs) == 0 {
		return errors.New("invalid managed client")
	}
	db := persistence.Client(ctx, p.store.client)
	grants := []string{"authorization_code"}
	for _, scope := range spec.Scopes {
		if scope == "offline_access" {
			grants = append(grants, "refresh_token")
		}
	}
	row, err := db.OAuthClient.Query().Where(oauthclient.ID(spec.ID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		if !spec.Public && len(hash) == 0 {
			return errors.New("confidential client credentials required")
		}
		return db.OAuthClient.Create().SetID(spec.ID).SetOwnerID(spec.OwnerID).SetName(spec.Name).SetPublic(spec.Public).SetEnabled(spec.Enabled).SetTestOnly(spec.TestOnly).SetRedirectUris(spec.RedirectURIs).SetPostLogoutRedirectUris(spec.PostLogoutRedirectURIs).SetGrantTypes(grants).SetResponseTypes([]string{"code"}).SetScopes(spec.Scopes).SetAudiences([]string{p.issuer + "/api"}).SetSecretHash(hash).Exec(ctx)
	}
	if err != nil {
		return err
	}
	if row.OwnerID != spec.OwnerID || row.OwnerID == "" || row.Public != spec.Public || len(hash) > 0 {
		return errors.New("managed client ownership mismatch")
	}
	return db.OAuthClient.UpdateOneID(spec.ID).SetName(spec.Name).SetEnabled(spec.Enabled).SetTestOnly(spec.TestOnly).SetRedirectUris(spec.RedirectURIs).SetPostLogoutRedirectUris(spec.PostLogoutRedirectURIs).SetGrantTypes(grants).SetResponseTypes([]string{"code"}).SetScopes(spec.Scopes).SetAudiences([]string{p.issuer + "/api"}).Exec(ctx)
}
func (p *Provider) SetManagedClientEnabled(ctx context.Context, id, owner string, enabled bool) error {
	db := persistence.Client(ctx, p.store.client)
	row, err := db.OAuthClient.Query().Where(oauthclient.ID(id)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	if row.OwnerID != owner || owner == "" {
		return ErrGrantMissing
	}
	if !enabled {
		ids, err := db.AuthorizationGrant.Query().Where(authorizationgrant.ClientID(id)).Order(ent.Asc(authorizationgrant.FieldID)).IDs(ctx)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, err := db.AuthorizationGrant.Query().Where(authorizationgrant.ID(id)).ForUpdate().Only(ctx); err != nil {
				return err
			}
			if err := db.AuthorizationGrant.UpdateOneID(id).SetRevoked(true).Exec(ctx); err != nil {
				return err
			}
			if _, err := db.ProtocolSession.Update().Where(protocolsession.RequestID(id)).SetRevoked(true).Save(ctx); err != nil {
				return err
			}
		}
	}
	return db.OAuthClient.UpdateOneID(id).SetEnabled(enabled).Exec(ctx)
}
func (p *Provider) RotateManagedSecret(ctx context.Context, id, owner string) (string, error) {
	secret, hash, err := NewClientSecret()
	if err != nil {
		return "", err
	}
	count, err := persistence.Client(ctx, p.store.client).OAuthClient.Update().Where(oauthclient.ID(id), oauthclient.OwnerID(owner), oauthclient.Public(false)).SetSecretHash(hash).Save(ctx)
	if err != nil {
		return "", err
	}
	if count != 1 {
		return "", ErrGrantMissing
	}
	return secret, nil
}
func (p *Provider) AllowSubject(ctx context.Context, clientID, userID string) error {
	return p.store.allowSubject(ctx, clientID, userID)
}
