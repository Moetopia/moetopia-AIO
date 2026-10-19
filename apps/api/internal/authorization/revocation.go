package authorization

import (
	"errors"
	"net/http"

	"github.com/ory/fosite"
)

func (p *Provider) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx, err := p.store.BeginTX(r.Context())
	if err != nil {
		p.oauth.WriteAccessError(r.Context(), w, nil, fosite.ErrTemporarilyUnavailable)
		return
	}
	defer p.store.Rollback(ctx)
	if err := r.ParseForm(); err != nil || r.PostForm.Get("token") == "" {
		p.oauth.WriteRevocationResponse(ctx, w, fosite.ErrInvalidRequest)
		return
	}
	err = p.oauth.NewRevocationRequest(ctx, r)
	if err != nil {
		if _, _, basic := r.BasicAuth(); basic && errors.Is(err, fosite.ErrInvalidClient) {
			w.Header().Set("WWW-Authenticate", `Basic realm="oauth"`)
		}
		if errors.Is(err, fosite.ErrInvalidClient) || errors.Is(err, fosite.ErrInvalidRequest) || errors.Is(err, fosite.ErrUnauthorizedClient) {
			p.oauth.WriteRevocationResponse(ctx, w, err)
		} else {
			p.oauth.WriteAccessError(ctx, w, nil, fosite.ErrTemporarilyUnavailable)
		}
		return
	}
	if err := p.store.Commit(ctx); err != nil {
		p.oauth.WriteAccessError(ctx, w, nil, fosite.ErrTemporarilyUnavailable)
		return
	}
	p.oauth.WriteRevocationResponse(ctx, w, nil)
}
