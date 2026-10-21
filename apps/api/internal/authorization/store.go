package authorization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/oauthclient"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
)

type Store struct {
	client   *ent.Client
	box      *cryptobox.Box
	strategy oauth2.CoreStrategy
}

func NewStore(client *ent.Client, box *cryptobox.Box) *Store { return &Store{client: client, box: box} }

func (s *Store) GetClient(ctx context.Context, id string) (_ fosite.Client, resultErr error) {
	defer func() { observeIntrospectionRead(ctx, resultErr) }()
	query := s.database(ctx).OAuthClient.Query().Where(oauthclient.ID(id))
	if state, ok := ctx.Value(transactionKey{}).(transaction); ok && !*state.clientRead {
		// Pin the incoming client's configuration before grant locks; foreign snapshots must not invert this order.
		*state.clientRead = true
		query.ForShare()
	}
	row, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !row.Enabled {
		return nil, fosite.ErrInvalidClient
	}
	return &fosite.DefaultClient{ID: row.ID, Secret: row.SecretHash, RedirectURIs: row.RedirectUris, GrantTypes: row.GrantTypes, ResponseTypes: row.ResponseTypes, Scopes: row.Scopes, Audience: row.Audiences, Public: row.Public}, nil
}

func (s *Store) ClientAssertionJWTValid(context.Context, string) error {
	return fosite.ErrInvalidClient.WithHint("private_key_jwt is not enabled")
}
func (s *Store) SetClientAssertionJWT(context.Context, string, time.Time) error {
	return fosite.ErrInvalidClient.WithHint("private_key_jwt is not enabled")
}

type requestSnapshot struct {
	ID                string
	ClientID          string
	RequestedAt       time.Time
	Scopes            fosite.Arguments
	Audience          fosite.Arguments
	Session           json.RawMessage
	RequestedScopes   fosite.Arguments
	RequestedAudience fosite.Arguments
	Form              url.Values
}

func signatureID(kind, signature string) string {
	sum := sha256.Sum256([]byte(kind + ":" + signature))
	return hex.EncodeToString(sum[:])
}

func (s *Store) CreateAccessTokenSession(ctx context.Context, signature string, request fosite.Requester) error {
	return s.save(ctx, "access", signature, request, fosite.AccessToken)
}

func (s *Store) save(ctx context.Context, kind, signature string, request fosite.Requester, tokenType fosite.TokenType) error {
	if kind == "code" || kind == "access" {
		if err := s.ensureGrant(ctx, request, tokenType); err != nil {
			return err
		}
	}
	session, err := json.Marshal(request.GetSession())
	if err != nil {
		return err
	}
	form := url.Values{}
	for _, name := range []string{"redirect_uri", "code_challenge", "code_challenge_method", "nonce", "prompt", "max_age", "response_type"} {
		if value := request.GetRequestForm().Get(name); value != "" {
			form.Set(name, value)
		}
	}
	payload, err := json.Marshal(requestSnapshot{ID: request.GetID(), ClientID: request.GetClient().GetID(), RequestedAt: request.GetRequestedAt(), Scopes: request.GetGrantedScopes(), Audience: request.GetGrantedAudience(), RequestedScopes: request.GetRequestedScopes(), RequestedAudience: request.GetRequestedAudience(), Form: form, Session: session})
	if err != nil {
		return err
	}
	id := signatureID(kind, signature)
	encrypted, err := s.box.Seal(payload, "protocol:"+id)
	if err != nil {
		return err
	}
	_, err = s.database(ctx).ProtocolSession.Create().SetID(id).SetKind(kind).SetRequestID(request.GetID()).SetClientID(request.GetClient().GetID()).SetPayload(encrypted).SetExpiresAt(request.GetSession().GetExpiresAt(tokenType)).Save(ctx)
	return err
}

func (s *Store) GetAccessTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.load(ctx, "access", signature, session)
}

func (s *Store) load(ctx context.Context, kind, signature string, session fosite.Session) (_ fosite.Requester, resultErr error) {
	defer func() { observeIntrospectionRead(ctx, resultErr) }()
	id := signatureID(kind, signature)
	query := s.database(ctx).ProtocolSession.Query().Where(protocolsession.ID(id))
	row, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	grant, err := s.grant(ctx, row.RequestID)
	if err != nil {
		return nil, err
	}
	row, err = query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	plaintext, err := s.box.Open(row.Payload, "protocol:"+id)
	if err != nil {
		return nil, errors.New("protocol state could not be decrypted")
	}
	var snapshot requestSnapshot
	if err := json.Unmarshal(plaintext, &snapshot); err != nil {
		return nil, err
	}
	if session == nil {
		session = newSession("")
	}
	if err := json.Unmarshal(snapshot.Session, session); err != nil {
		return nil, err
	}
	client, err := s.GetClient(ctx, snapshot.ClientID)
	if err != nil {
		return nil, err
	}
	request := &fosite.Request{ID: snapshot.ID, RequestedAt: snapshot.RequestedAt, Client: client, GrantedScope: snapshot.Scopes, RequestedScope: snapshot.RequestedScopes, GrantedAudience: snapshot.Audience, RequestedAudience: snapshot.RequestedAudience, Form: snapshot.Form, Session: session}
	if kind == "code" || kind == "refresh" {
		if incoming, ok := ctx.Value(fosite.AccessRequestContextKey).(fosite.AccessRequester); ok {
			if incoming.GetClient().GetID() != client.GetID() || !s.authentic(ctx, kind, request, incoming.GetRequestForm().Get(map[string]string{"code": "code", "refresh": "refresh_token"}[kind])) {
				return nil, fosite.ErrNotFound
			}
		}
	}
	if raw, ok := ctx.Value(fosite.RequestContextKey).(*http.Request); ok && raw.URL.Path == "/oauth/revoke" && raw.PostForm.Get("token") != "" {
		if !s.authentic(ctx, kind, request, raw.PostForm.Get("token")) {
			return nil, fosite.ErrNotFound
		}
		if kind == "refresh" && !grant.Revoked && time.Now().Before(grant.ExpiresAt) {
			return request, nil
		}
	}
	if grant.Revoked || !time.Now().Before(grant.ExpiresAt) {
		if kind == "code" {
			return nil, fosite.ErrNotFound
		}
		if kind == "refresh" && !grant.Revoked {
			return nil, fosite.ErrNotFound
		}
		return request, fosite.ErrInactiveToken
	}
	if kind == "code" && row.Revoked {
		return request, fosite.ErrInvalidatedAuthorizeCode
	}
	if row.Revoked {
		return request, fosite.ErrInactiveToken
	}
	if !time.Now().Before(row.ExpiresAt) {
		return nil, fosite.ErrNotFound
	}
	return request, nil
}

func (s *Store) authentic(ctx context.Context, kind string, request fosite.Requester, raw string) bool {
	if s.strategy == nil {
		return false
	}
	probe := request.Sanitize([]string{})
	probe.SetSession(request.GetSession().Clone())
	// Spent credentials still need signature verification before revoking their family.
	switch kind {
	case "code":
		probe.GetSession().SetExpiresAt(fosite.AuthorizeCode, time.Now().Add(time.Minute))
		return s.strategy.ValidateAuthorizeCode(ctx, probe, raw) == nil
	case "refresh":
		probe.GetSession().SetExpiresAt(fosite.RefreshToken, time.Now().Add(time.Minute))
		return s.strategy.ValidateRefreshToken(ctx, probe, raw) == nil
	case "access":
		return s.strategy.ValidateAccessToken(ctx, probe, raw) == nil
	}
	return false
}

func (s *Store) DeleteAccessTokenSession(ctx context.Context, signature string) error {
	_, err := s.database(ctx).ProtocolSession.Update().Where(protocolsession.ID(signatureID("access", signature))).SetRevoked(true).Save(ctx)
	return err
}

var _ fosite.Storage = (*Store)(nil)
var _ oauth2.AccessTokenStorage = (*Store)(nil)
