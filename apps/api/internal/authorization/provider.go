package authorization

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/token/jwt"
	"moetopia.local/aio/api/internal/identity"
)

type Provider struct {
	oauth  fosite.OAuth2Provider
	key    *Key
	issuer string
	store  *Store
	users  SubjectSource
}

type SubjectSource interface {
	User(context.Context, string) (identity.User, error)
}

func NewProvider(store *Store, key *Key, issuer, encryptionKey string, users SubjectSource) (*Provider, error) {
	root, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, root)
	_, _ = mac.Write([]byte("moetopia:oauth:opaque-token:v1"))
	cfg := &fosite.Config{AccessTokenLifespan: 10 * time.Minute, RefreshTokenLifespan: refreshIdleLifetime, AuthorizeCodeLifespan: 5 * time.Minute, IDTokenLifespan: 10 * time.Minute, IDTokenIssuer: issuer, AccessTokenIssuer: issuer, GlobalSecret: mac.Sum(nil), ScopeStrategy: fosite.ExactScopeStrategy, EnforcePKCE: true, RefreshTokenScopes: []string{"offline_access"}, SendDebugMessagesToClients: false, JWTScopeClaimKey: jwt.JWTScopeFieldString}
	strategy := compose.NewOAuth2JWTStrategy(func(context.Context) (interface{}, error) { return key.Private, nil }, compose.NewOAuth2HMACStrategy(cfg), cfg)
	common := &compose.CommonStrategy{CoreStrategy: strategy, OpenIDConnectTokenStrategy: compose.NewOpenIDConnectStrategy(func(context.Context) (interface{}, error) { return key.Private, nil }, cfg), Signer: strategy.Signer}
	store.strategy = strategy
	provider := compose.Compose(cfg, store, common, compose.OAuth2AuthorizeExplicitFactory, compose.OAuth2RefreshTokenGrantFactory, compose.OAuth2ClientCredentialsGrantFactory, compose.OpenIDConnectExplicitFactory, compose.OpenIDConnectRefreshFactory, compose.OAuth2PKCEFactory, compose.OAuth2TokenIntrospectionFactory, compose.OAuth2TokenRevocationFactory)
	return &Provider{oauth: provider, key: key, issuer: issuer, store: store, users: users}, nil
}

func (p *Provider) Token(w http.ResponseWriter, r *http.Request) {
	ctx, err := p.store.BeginTX(r.Context())
	if err != nil {
		p.oauth.WriteAccessError(r.Context(), w, nil, fosite.ErrTemporarilyUnavailable)
		return
	}
	defer p.store.Rollback(ctx)
	session := newSession(p.key.ID)
	request, err := p.oauth.NewAccessRequest(ctx, r, session)
	if err != nil {
		if _, _, basic := r.BasicAuth(); basic && errors.Is(err, fosite.ErrInvalidClient) {
			w.Header().Set("WWW-Authenticate", `Basic realm="oauth"`)
		}
		if state := ctx.Value(transactionKey{}).(transaction); *state.replayed {
			if commitErr := p.store.Commit(ctx); commitErr != nil {
				err = fosite.ErrTemporarilyUnavailable
			}
		}
		p.oauth.WriteAccessError(ctx, w, request, err)
		return
	}
	if err := p.prepareToken(ctx, request); err != nil {
		p.oauth.WriteAccessError(ctx, w, request, err)
		return
	}
	response, err := p.oauth.NewAccessResponse(ctx, request)
	if err != nil {
		p.oauth.WriteAccessError(ctx, w, request, err)
		return
	}
	if err := p.store.Commit(ctx); err != nil {
		p.oauth.WriteAccessError(ctx, w, request, fosite.ErrTemporarilyUnavailable)
		return
	}
	p.oauth.WriteAccessResponse(ctx, w, request, response)
}

func (p *Provider) JWKS() any { return map[string]any{"keys": []map[string]string{p.key.PublicJWK()}} }
