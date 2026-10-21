package authorization

import (
	"context"
	"errors"

	"github.com/go-jose/go-jose/v3"
	"github.com/ory/fosite/token/jwt"
)

type rotatingSigner struct {
	*jwt.DefaultSigner
	keys keySource
}

func (s *rotatingSigner) Generate(ctx context.Context, claims jwt.MapClaims, header jwt.Mapper) (string, string, error) {
	key, err := pinnedKey(ctx)
	if err != nil {
		return "", "", err
	}
	if header == nil {
		return "", "", errors.New("token header unavailable")
	}
	headers := header.ToMap()
	headers["kid"] = key.ID
	return s.DefaultSigner.Generate(ctx, claims, &jwt.Headers{Extra: headers})
}

func (s *rotatingSigner) Decode(ctx context.Context, raw string) (*jwt.Token, error) {
	if len(raw) > 32768 {
		return nil, errors.New("invalid token size")
	}
	return jwt.ParseWithClaims(raw, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		id, ok := token.Header["kid"].(string)
		if !ok || len(id) > 128 || id == "" || token.Method != jose.RS256 {
			return nil, errors.New("invalid signing key header")
		}
		return s.keys.Public(ctx, id)
	})
}
func (s *rotatingSigner) Validate(ctx context.Context, raw string) (string, error) {
	if _, err := s.Decode(ctx, raw); err != nil {
		return "", err
	}
	return s.DefaultSigner.GetSignature(ctx, raw)
}
