package authorization

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/ory/fosite"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/authorizationinteraction"
	"moetopia.local/aio/api/internal/persistence/ent/protocolevent"
	"moetopia.local/aio/api/internal/sessions"
)

type AuthorizationDetails struct {
	ClientID      string    `json:"client_id"`
	ClientName    string    `json:"client_name"`
	RedirectURI   string    `json:"redirect_uri"`
	Scopes        []string  `json:"scopes"`
	Audiences     []string  `json:"audiences"`
	ExpiresAt     time.Time `json:"expires_at"`
	LoginRequired bool      `json:"login_required"`
}

func browserHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (p *Provider) beginInteraction(ctx context.Context, token string, query url.Values) (string, error) {
	id, err := sessions.RandomToken()
	if err != nil {
		return "", err
	}
	plain, err := json.Marshal(query)
	if err != nil {
		return "", err
	}
	payload, err := p.store.box.Seal(plain, "authorization:"+id)
	if err != nil {
		return "", err
	}
	_, err = p.store.database(ctx).AuthorizationInteraction.Create().SetID(id).SetClientID(query.Get("client_id")).SetBrowserHash(browserHash(token)).SetPayload(payload).SetExpiresAt(time.Now().Add(10 * time.Minute)).Save(ctx)
	return id, err
}
func (p *Provider) interaction(ctx context.Context, id, token string) (*ent.AuthorizationInteraction, url.Values, error) {
	if !sessions.ValidToken(id) || !sessions.ValidToken(token) {
		return nil, nil, ErrInteraction
	}
	row, err := p.store.database(ctx).AuthorizationInteraction.Get(ctx, id)
	if ent.IsNotFound(err) {
		return nil, nil, ErrInteraction
	}
	if err != nil {
		return nil, nil, err
	}
	if row.Consumed || !time.Now().Before(row.ExpiresAt) || subtle.ConstantTimeCompare([]byte(row.BrowserHash), []byte(browserHash(token))) != 1 {
		return nil, nil, ErrInteraction
	}
	plain, err := p.store.box.Open(row.Payload, "authorization:"+id)
	if err != nil {
		return nil, nil, err
	}
	query := url.Values{}
	if err := json.Unmarshal(plain, &query); err != nil {
		return nil, nil, err
	}
	return row, query, nil
}
func (p *Provider) Details(ctx context.Context, id, token string, authenticated time.Time) (AuthorizationDetails, error) {
	row, query, err := p.interaction(ctx, id, token)
	if err != nil {
		return AuthorizationDetails{}, err
	}
	ar, err := p.authorizeRequest(ctx, query)
	if err != nil {
		return AuthorizationDetails{}, ErrInteraction
	}
	client, err := p.store.database(ctx).OAuthClient.Get(ctx, row.ClientID)
	if err != nil {
		return AuthorizationDetails{}, err
	}
	return AuthorizationDetails{ClientID: row.ClientID, ClientName: client.Name, RedirectURI: query.Get("redirect_uri"), Scopes: ar.GetRequestedScopes(), Audiences: ar.GetRequestedAudience(), ExpiresAt: row.ExpiresAt, LoginRequired: loginRequired(query, row.CreatedAt, authenticated)}, nil
}

type redirectWriter struct{ header http.Header }

func (w *redirectWriter) Header() http.Header       { return w.header }
func (*redirectWriter) Write(p []byte) (int, error) { return len(p), nil }
func (*redirectWriter) WriteHeader(int)             {}

func (p *Provider) Decide(ctx context.Context, id, token string, user identity.User, authenticated time.Time, approve bool) (string, error) {
	ctx, err := p.store.BeginTX(ctx)
	if err != nil {
		return "", err
	}
	defer p.store.Rollback(ctx)
	row, query, err := p.interaction(ctx, id, token)
	if err != nil {
		return "", err
	}
	if loginRequired(query, row.CreatedAt, authenticated) {
		return "", ErrLoginRequired
	}
	ar, err := p.authorizeRequest(ctx, query)
	if err != nil {
		return "", ErrInteraction
	}
	n, err := p.store.database(ctx).AuthorizationInteraction.Update().Where(authorizationinteraction.ID(id), authorizationinteraction.Consumed(false), authorizationinteraction.ExpiresAtGT(time.Now()), authorizationinteraction.BrowserHash(browserHash(token))).SetConsumed(true).Save(ctx)
	if err != nil {
		return "", err
	}
	if n != 1 {
		return "", ErrInteraction
	}
	w := &redirectWriter{header: http.Header{}}
	if approve {
		if err := p.AllowSubject(ctx, ar.GetClient().GetID(), user.ID); err != nil {
			if errors.Is(err, ErrGrantMissing) {
				return "", ErrInteraction
			}
			return "", err
		}
		for _, scope := range ar.GetRequestedScopes() {
			ar.GrantScope(scope)
		}
		for _, audience := range ar.GetRequestedAudience() {
			ar.GrantAudience(audience)
		}
		session := newSession("")
		session.Subject, session.JWTClaims.Subject = user.ID, user.ID
		session.SecurityVersion = user.SecurityVersion
		session.JWTClaims.Extra = map[string]interface{}{"client_id": ar.GetClient().GetID()}
		session.IdentityClaims.Subject, session.IdentityClaims.AuthTime, session.IdentityClaims.RequestedAt = user.ID, authenticated, row.CreatedAt
		session.IdentityClaims.Extra = identityClaims(user, ar.GetGrantedScopes())
		session.IdentityClaims.Extra["security_version"] = user.SecurityVersion
		response, err := p.oauth.NewAuthorizeResponse(ctx, ar, session)
		if err != nil {
			return "", err
		}
		if err := p.store.recordProtocolEvent(ctx, ar.GetClient().GetID(), protocolevent.KindAuthorization); err != nil {
			return "", err
		}
		p.oauth.WriteAuthorizeResponse(ctx, w, ar, response)
	} else {
		p.oauth.WriteAuthorizeError(ctx, w, ar, fosite.ErrAccessDenied)
	}
	location := w.header.Get("Location")
	if location == "" {
		return "", ErrInteraction
	}
	if err := p.store.Commit(ctx); err != nil {
		return "", err
	}
	return location, nil
}
