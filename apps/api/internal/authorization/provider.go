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
	cfg := &fosite.Config{AccessTokenLifespan: 10 * time.Minute, AuthorizeCodeLifespan: 5 * time.Minute, IDTokenLifespan: 10 * time.Minute, IDTokenIssuer: issuer, AccessTokenIssuer: issuer, GlobalSecret: mac.Sum(nil), ScopeStrategy: fosite.ExactScopeStrategy, EnforcePKCE: true, RefreshTokenScopes: []string{"offline_access"}, SendDebugMessagesToClients: false, JWTScopeClaimKey: jwt.JWTScopeFieldString}
	strategy := compose.NewOAuth2JWTStrategy(func(context.Context) (interface{}, error) { return key.Private, nil }, compose.NewOAuth2HMACStrategy(cfg), cfg)
	common := &compose.CommonStrategy{CoreStrategy: strategy, OpenIDConnectTokenStrategy: compose.NewOpenIDConnectStrategy(func(context.Context) (interface{}, error) { return key.Private, nil }, cfg), Signer: strategy.Signer}
	provider := compose.Compose(cfg, store, common, compose.OAuth2AuthorizeExplicitFactory, compose.OAuth2ClientCredentialsGrantFactory, compose.OpenIDConnectExplicitFactory, compose.OAuth2PKCEFactory, compose.OAuth2TokenIntrospectionFactory)
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
		if state := ctx.Value(transactionKey{}).(transaction); *state.replayed {
			if commitErr := p.store.Commit(ctx); commitErr != nil {
				err = fosite.ErrTemporarilyUnavailable
			}
		}
		p.oauth.WriteAccessError(ctx, w, request, err)
		return
	}
	if request.GetRequestedScopes().Has("offline_access") {
		p.oauth.WriteAccessError(ctx, w, request, fosite.ErrInvalidScope)
		return
	}
	if request.GetGrantTypes().ExactOne("authorization_code") {
		if p.users == nil {
			p.oauth.WriteAccessError(ctx, w, request, fosite.ErrTemporarilyUnavailable)
			return
		}
		if _, err := p.users.User(ctx, session.Subject); err != nil {
			failure := fosite.ErrTemporarilyUnavailable
			if errors.Is(err, identity.ErrMissing) {
				failure = fosite.ErrInvalidGrant
			}
			p.oauth.WriteAccessError(ctx, w, request, failure)
			return
		}
	} else {
		if request.GetRequestedScopes().Has("openid") || request.GetRequestedScopes().Has("profile") || request.GetRequestedScopes().Has("email") {
			p.oauth.WriteAccessError(ctx, w, request, fosite.ErrInvalidScope)
			return
		}
		for _, scope := range request.GetRequestedScopes() {
			request.GrantScope(scope)
		}
		audiences := request.GetRequestedAudience()
		if len(audiences) == 0 && len(request.GetClient().GetAudience()) == 1 {
			audiences = request.GetClient().GetAudience()
		}
		if len(audiences) != 1 {
			p.oauth.WriteAccessError(ctx, w, request, fosite.ErrInvalidRequest.WithHint("Exactly one API audience is required"))
			return
		}
		if err := fosite.DefaultAudienceMatchingStrategy(request.GetClient().GetAudience(), audiences); err != nil {
			p.oauth.WriteAccessError(ctx, w, request, err)
			return
		}
		request.GrantAudience(audiences[0])
		session.Subject = request.GetClient().GetID()
		session.JWTClaims.Subject = session.Subject
		session.JWTClaims.Extra = map[string]interface{}{"client_id": request.GetClient().GetID()}
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
