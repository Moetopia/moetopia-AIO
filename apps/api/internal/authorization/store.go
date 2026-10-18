package authorization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/protocolsession"
)

type Store struct {
	client *ent.Client
	box    *cryptobox.Box
}

func NewStore(client *ent.Client, box *cryptobox.Box) *Store { return &Store{client: client, box: box} }

func (s *Store) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	row, err := s.database(ctx).OAuthClient.Get(ctx, id)
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

func (s *Store) load(ctx context.Context, kind, signature string, session fosite.Session) (fosite.Requester, error) {
	id := signatureID(kind, signature)
	query := s.database(ctx).ProtocolSession.Query().Where(protocolsession.ID(id))
	if kind == "code" {
		if _, ok := ctx.Value(transactionKey{}).(transaction); ok {
			query.ForUpdate()
		}
	}
	row, err := query.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if kind != "code" && row.Revoked || !row.Revoked && !time.Now().Before(row.ExpiresAt) {
		return nil, fosite.ErrInactiveToken
	}
	plaintext, err := s.box.Open(row.Payload, "protocol:"+id)
	if err != nil {
		return nil, errors.New("protocol state could not be decrypted")
	}
	var snapshot requestSnapshot
	if err := json.Unmarshal(plaintext, &snapshot); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(snapshot.Session, session); err != nil {
		return nil, err
	}
	client, err := s.GetClient(ctx, snapshot.ClientID)
	if err != nil {
		return nil, err
	}
	request := &fosite.Request{ID: snapshot.ID, RequestedAt: snapshot.RequestedAt, Client: client, GrantedScope: snapshot.Scopes, RequestedScope: snapshot.RequestedScopes, GrantedAudience: snapshot.Audience, RequestedAudience: snapshot.RequestedAudience, Form: snapshot.Form, Session: session}
	if kind == "code" && row.Revoked {
		return request, fosite.ErrInvalidatedAuthorizeCode
	}
	return request, nil
}

func (s *Store) DeleteAccessTokenSession(ctx context.Context, signature string) error {
	_, err := s.database(ctx).ProtocolSession.Update().Where(protocolsession.ID(signatureID("access", signature))).SetRevoked(true).Save(ctx)
	return err
}

var _ fosite.Storage = (*Store)(nil)
var _ oauth2.AccessTokenStorage = (*Store)(nil)
