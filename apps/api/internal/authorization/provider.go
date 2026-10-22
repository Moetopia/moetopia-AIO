package authorization

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/token/jwt"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence/ent/protocolevent"
)

type Provider struct {
	oauth  fosite.OAuth2Provider
	keys   keySource
	issuer string
	store  *Store
	users  SubjectSource
}

type SubjectSource interface {
	User(context.Context, string) (identity.User, error)
}

func NewProvider(store *Store, key *Key, issuer, encryptionKey string, users SubjectSource) (*Provider, error) {
	return newProvider(store, fixedKey{key}, issuer, encryptionKey, users)
}

func NewPersistentProvider(store *Store, issuer, encryptionKey string, users SubjectSource) (*Provider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keys := persistentKeys{store}
	_, err := keys.Signing(ctx)
	if err != nil {
		return nil, err
	}
	return newProvider(store, keys, issuer, encryptionKey, users)
}

func newProvider(store *Store, keys keySource, issuer, encryptionKey string, users SubjectSource) (*Provider, error) {
	root, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, root)
	_, _ = mac.Write([]byte("moetopia:oauth:opaque-token:v1"))
	cfg := &fosite.Config{AccessTokenLifespan: 10 * time.Minute, RefreshTokenLifespan: refreshIdleLifetime, AuthorizeCodeLifespan: 5 * time.Minute, IDTokenLifespan: 10 * time.Minute, IDTokenIssuer: issuer, AccessTokenIssuer: issuer, GlobalSecret: mac.Sum(nil), ScopeStrategy: fosite.ExactScopeStrategy, EnforcePKCE: true, RefreshTokenScopes: []string{"offline_access"}, SendDebugMessagesToClients: false, JWTScopeClaimKey: jwt.JWTScopeFieldString}
	getter := func(ctx context.Context) (interface{}, error) {
		key, err := pinnedKey(ctx)
		if err != nil {
			return nil, err
		}
		return key.Private, nil
	}
	signer := &rotatingSigner{DefaultSigner: &jwt.DefaultSigner{GetPrivateKey: getter}, keys: keys}
	strategy := compose.NewOAuth2JWTStrategy(getter, compose.NewOAuth2HMACStrategy(cfg), cfg)
	strategy.Signer = signer
	openid := compose.NewOpenIDConnectStrategy(getter, cfg)
	openid.Signer = signer
	common := &compose.CommonStrategy{CoreStrategy: strategy, OpenIDConnectTokenStrategy: openid, Signer: signer}
	store.strategy = strategy
	provider := compose.Compose(cfg, store, common, compose.OAuth2AuthorizeExplicitFactory, compose.OAuth2RefreshTokenGrantFactory, compose.OAuth2ClientCredentialsGrantFactory, compose.OpenIDConnectExplicitFactory, compose.OpenIDConnectRefreshFactory, compose.OAuth2PKCEFactory, compose.OAuth2TokenIntrospectionFactory, compose.OAuth2TokenRevocationFactory)
	return &Provider{oauth: provider, keys: keys, issuer: issuer, store: store, users: users}, nil
}

func (p *Provider) Token(w http.ResponseWriter, r *http.Request) {
	ctx, err := p.store.BeginTX(r.Context())
	if err != nil {
		p.oauth.WriteAccessError(r.Context(), w, nil, fosite.ErrTemporarilyUnavailable)
		return
	}
	defer p.store.Rollback(ctx)
	key, err := p.keys.Signing(ctx)
	if err != nil {
		p.oauth.WriteAccessError(ctx, w, nil, fosite.ErrTemporarilyUnavailable)
		return
	}
	ctx = context.WithValue(ctx, signingKeyContext{}, key)
	session := newSession(key.ID)
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
	kind := protocolevent.KindCodeExchange
	if request.GetGrantTypes().ExactOne("refresh_token") {
		kind = protocolevent.KindRefresh
	}
	if request.GetGrantTypes().ExactOne("client_credentials") {
		kind = protocolevent.KindMachineToken
	}
	if err := p.store.recordProtocolEvent(ctx, request.GetClient().GetID(), kind); err != nil {
		p.oauth.WriteAccessError(ctx, w, request, fosite.ErrTemporarilyUnavailable)
		return
	}
	if err := p.store.Commit(ctx); err != nil {
		p.oauth.WriteAccessError(ctx, w, request, fosite.ErrTemporarilyUnavailable)
		return
	}
	p.oauth.WriteAccessResponse(ctx, w, request, response)
}

func (p *Provider) JWKS(ctx context.Context) (any, error) {
	keys, err := p.keys.List(ctx)
	return map[string]any{"keys": keys}, err
}

func (p *Provider) WriteJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	keys, err := p.JWKS(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "temporarily_unavailable"})
		return
	}
	_ = json.NewEncoder(w).Encode(keys)
}
