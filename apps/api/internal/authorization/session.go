package authorization

import (
	"github.com/mohae/deepcopy"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/token/jwt"
)

type Session struct {
	oauth2.JWTSession
	IdentityClaims  *jwt.IDTokenClaims `json:"id_token_claims"`
	IdentityHeaders *jwt.Headers       `json:"id_token_headers"`
	SecurityVersion int                `json:"security_version"`
}

func newSession(kid string) *Session {
	return &Session{
		JWTSession:      oauth2.JWTSession{JWTClaims: &jwt.JWTClaims{}, JWTHeader: &jwt.Headers{Extra: map[string]interface{}{"typ": "at+jwt", "kid": kid}}},
		IdentityClaims:  &jwt.IDTokenClaims{},
		IdentityHeaders: &jwt.Headers{Extra: map[string]interface{}{"typ": "JWT", "kid": kid}},
	}
}
func (s *Session) IDTokenClaims() *jwt.IDTokenClaims { return s.IdentityClaims }
func (s *Session) IDTokenHeaders() *jwt.Headers      { return s.IdentityHeaders }
func (s *Session) Clone() fosite.Session             { return deepcopy.Copy(s).(*Session) }

var _ openid.Session = (*Session)(nil)
var _ oauth2.JWTSessionContainer = (*Session)(nil)
