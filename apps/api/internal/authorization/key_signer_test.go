package authorization

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/ory/fosite/token/jwt"
)

func TestRotatingSignerPinsHeaderAndValidatesClaims(t *testing.T) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	key := &Key{ID: "current-key", Private: private}
	ctx := context.WithValue(t.Context(), signingKeyContext{}, key)
	signer := &rotatingSigner{DefaultSigner: &jwt.DefaultSigner{GetPrivateKey: func(context.Context) (interface{}, error) { return private, nil }}, keys: fixedKey{key}}
	raw, _, err := signer.Generate(ctx, jwt.MapClaims{"sub": "test-user", "iss": "https://account.example.test", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix()}, &jwt.Headers{Extra: map[string]interface{}{"kid": "old-key"}})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := signer.Decode(t.Context(), raw)
	if err != nil {
		t.Fatalf("signed token failed verification: %v", err)
	}
	if parsed.Header["kid"] != key.ID || parsed.Claims["sub"] != "test-user" {
		t.Fatal("inconsistent signing header or claims")
	}
}
