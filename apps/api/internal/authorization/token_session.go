package authorization

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ory/fosite"
	"moetopia.local/aio/api/internal/identity"
)

func (p *Provider) prepareToken(ctx context.Context, request fosite.AccessRequester) error {
	session, ok := request.GetSession().(*Session)
	if !ok {
		return fosite.ErrServerError
	}
	key, err := pinnedKey(ctx)
	if err != nil {
		return fosite.ErrTemporarilyUnavailable
	}
	session.JWTHeader.Extra = map[string]interface{}{"typ": "at+jwt", "kid": key.ID}
	session.IdentityHeaders.Extra = map[string]interface{}{"typ": "JWT", "kid": key.ID}
	if request.GetGrantTypes().ExactOne("client_credentials") {
		for _, scope := range []string{"openid", "profile", "email", "offline_access"} {
			if request.GetRequestedScopes().Has(scope) {
				return fosite.ErrInvalidScope
			}
		}
		for _, scope := range request.GetRequestedScopes() {
			request.GrantScope(scope)
		}
		audiences := request.GetRequestedAudience()
		if len(audiences) == 0 && len(request.GetClient().GetAudience()) == 1 {
			audiences = request.GetClient().GetAudience()
		}
		if len(audiences) != 1 {
			return fosite.ErrInvalidRequest.WithHint("Exactly one API audience is required")
		}
		if err := fosite.DefaultAudienceMatchingStrategy(request.GetClient().GetAudience(), audiences); err != nil {
			return err
		}
		request.GrantAudience(audiences[0])
		session.Subject = request.GetClient().GetID()
		session.JWTClaims.Subject = session.Subject
		session.JWTClaims.Extra = map[string]interface{}{"client_id": request.GetClient().GetID()}
		return nil
	}
	if p.users == nil {
		return fosite.ErrTemporarilyUnavailable
	}
	user, err := p.users.User(ctx, session.Subject)
	if errors.Is(err, identity.ErrMissing) {
		return fosite.ErrInvalidGrant
	}
	if err != nil {
		return fosite.ErrTemporarilyUnavailable
	}
	if user.SecurityVersion != session.SecurityVersion {
		return fosite.ErrInvalidGrant
	}
	if err := p.AllowSubject(ctx, request.GetClient().GetID(), session.Subject); err != nil {
		if errors.Is(err, ErrGrantMissing) {
			return fosite.ErrInvalidGrant
		}
		return fosite.ErrTemporarilyUnavailable
	}
	grant, err := p.store.grant(ctx, request.GetID())
	if err != nil {
		return fosite.ErrTemporarilyUnavailable
	}
	if grant.Revoked || !time.Now().Before(grant.ExpiresAt) {
		return fosite.ErrInvalidGrant
	}
	if request.GetGrantTypes().ExactOne("refresh_token") {
		if value := request.GetRequestForm().Get("scope"); value != "" {
			scopes := fosite.Arguments(strings.Fields(value))
			if len(scopes) != len(request.GetGrantedScopes()) {
				return fosite.ErrInvalidScope.WithHint("Refresh scope must be omitted or equal the approved scope set")
			}
			for _, scope := range request.GetGrantedScopes() {
				if !scopes.Has(scope) {
					return fosite.ErrInvalidScope
				}
			}
		}
		if audience := fosite.GetAudiences(request.GetRequestForm()); len(audience) > 0 && (len(audience) != 1 || !request.GetGrantedAudience().Has(audience[0])) {
			return fosite.ErrInvalidRequest
		}
		session.IdentityClaims.Extra = identityClaims(user, request.GetGrantedScopes())
		session.IdentityClaims.Extra["security_version"] = user.SecurityVersion
	}
	for _, token := range []fosite.TokenType{fosite.AccessToken, fosite.RefreshToken} {
		expires := session.GetExpiresAt(token)
		if !expires.IsZero() && expires.After(grant.ExpiresAt) {
			session.SetExpiresAt(token, grant.ExpiresAt)
		}
	}
	return nil
}
