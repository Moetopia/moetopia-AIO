package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/ory/fosite"
	"moetopia.local/aio/api/internal/identity"
)

type introspectionReadKey struct{}

func observeIntrospectionRead(ctx context.Context, err error) {
	failed, ok := ctx.Value(introspectionReadKey{}).(*bool)
	if ok && err != nil && !errors.Is(err, fosite.ErrNotFound) && !errors.Is(err, fosite.ErrInactiveToken) && !errors.Is(err, fosite.ErrInvalidClient) {
		*failed = true
	}
}

func (p *Provider) Introspect(w http.ResponseWriter, r *http.Request) {
	fail := func(err error) { p.oauth.WriteAccessError(r.Context(), w, nil, err) }
	unauthorized := func() {
		w.Header().Set("WWW-Authenticate", `Basic realm="oauth"`)
		p.oauth.WriteAccessError(r.Context(), w, nil, fosite.ErrInvalidClient)
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if r.Method != http.MethodPost || err != nil || mediaType != "application/x-www-form-urlencoded" {
		fail(fosite.ErrInvalidRequest)
		return
	}
	if err := r.ParseForm(); err != nil || len(r.PostForm["token"]) != 1 || strings.TrimSpace(r.PostForm.Get("token")) == "" || len(r.PostForm["token_type_hint"]) > 1 {
		fail(fosite.ErrInvalidRequest)
		return
	}
	id, _, basic := r.BasicAuth()
	id, err = url.QueryUnescape(id)
	// Fosite also accepts bearer authorization; this endpoint only admits confidential clients using Basic.
	if !basic || err != nil || r.Form.Get("access_token") != "" || r.Form.Get("client_id") != "" || r.Form.Get("client_secret") != "" {
		unauthorized()
		return
	}
	ctx, err := p.store.BeginTX(r.Context())
	if err != nil {
		fail(fosite.ErrTemporarilyUnavailable)
		return
	}
	defer p.store.Rollback(ctx)
	client, err := p.store.GetClient(ctx, id)
	if errors.Is(err, fosite.ErrNotFound) || errors.Is(err, fosite.ErrInvalidClient) || err == nil && client.IsPublic() {
		unauthorized()
		return
	}
	if err != nil {
		fail(fosite.ErrTemporarilyUnavailable)
		return
	}
	failed := false
	ctx = context.WithValue(ctx, introspectionReadKey{}, &failed)
	result, err := p.oauth.NewIntrospectionRequest(ctx, r, newSession(p.key.ID))
	if failed {
		fail(fosite.ErrTemporarilyUnavailable)
		return
	}
	if err != nil {
		if errors.Is(err, fosite.ErrRequestUnauthorized) && !errors.Is(err, fosite.ErrInactiveToken) {
			unauthorized()
		} else {
			p.oauth.WriteIntrospectionError(ctx, w, err)
		}
		return
	}
	request := result.GetAccessRequester()
	inactive := func() { p.oauth.WriteIntrospectionError(ctx, w, fosite.ErrInactiveToken) }
	if request.GetClient().GetID() != client.GetID() {
		inactive()
		return
	}
	grant, err := p.store.grant(ctx, request.GetID())
	if err != nil {
		fail(fosite.ErrTemporarilyUnavailable)
		return
	}
	session, ok := request.GetSession().(*Session)
	if !ok {
		fail(fosite.ErrTemporarilyUnavailable)
		return
	}
	if grant.UserID != "" || session.GetSubject() != client.GetID() || session.IdentityClaims != nil && session.IdentityClaims.Subject != "" {
		if p.users == nil {
			fail(fosite.ErrTemporarilyUnavailable)
			return
		}
		user, err := p.users.User(ctx, session.GetSubject())
		if errors.Is(err, identity.ErrMissing) {
			inactive()
			return
		}
		if err != nil {
			fail(fosite.ErrTemporarilyUnavailable)
			return
		}
		if user.SecurityVersion != session.SecurityVersion || grant.UserID != "" && grant.UserID != user.ID {
			inactive()
			return
		}
	}
	response := map[string]any{"active": true, "client_id": client.GetID(), "iss": p.issuer, "scope": strings.Join(request.GetGrantedScopes(), " "), "sub": request.GetSession().GetSubject()}
	if expiry := request.GetSession().GetExpiresAt(result.GetTokenUse()); !expiry.IsZero() {
		response["exp"] = expiry.Unix()
	}
	if len(request.GetGrantedAudience()) > 0 {
		response["aud"] = request.GetGrantedAudience()
	}
	if result.GetTokenUse() == fosite.AccessToken {
		response["token_type"] = fosite.BearerAccessToken
	}
	if err := p.store.Commit(ctx); err != nil {
		fail(fosite.ErrTemporarilyUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_ = json.NewEncoder(w).Encode(response)
}
