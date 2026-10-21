package httpserver

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"moetopia.local/aio/api/internal/relyingparty"
)

type RPLogoutOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      struct {
		RedirectTo string `json:"redirect_to"`
	}
}

func registerRPLogout(api huma.API, path string, rp *relyingparty.Service) {
	huma.Post(api, path, func(ctx context.Context, input *AccountWrite) (*RPLogoutOutput, error) {
		if rp == nil {
			return nil, huma.Error503ServiceUnavailable("RP logout unavailable")
		}
		token := rp.CookieToken(input.Cookie)
		if !rp.Sessions.Protect(token, input.Origin, input.CSRF) {
			return nil, huma.Error403Forbidden("Request verification failed")
		}
		if _, _, err := rp.User(ctx, token); err != nil {
			return nil, accountError(err)
		}
		started, err := rp.BeginLogout(ctx, token, input.Cookie)
		if err != nil {
			return nil, accountError(err)
		}
		result := &RPLogoutOutput{SetCookie: started.Cookies}
		result.Body.RedirectTo = started.RedirectTo
		return result, nil
	})
}
