package password

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

var capacity = make(chan struct{}, 2)

const prefix = "$argon2id$v=19$m=65536,t=3,p=1$"

func Valid(value string) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) >= 15 && len(value) <= 256
}

func derive(ctx context.Context, value string, salt []byte) ([]byte, error) {
	select {
	case capacity <- struct{}{}:
		defer func() { <-capacity }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return argon2.IDKey([]byte(value), salt, 3, 64*1024, 1, 32), nil
}

func Hash(ctx context.Context, value string) (string, error) {
	if !Valid(value) {
		return "", errors.New("password must contain at least 15 characters and at most 256 bytes")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := derive(ctx, value, salt)
	if err != nil {
		return "", err
	}
	return prefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

func Verify(ctx context.Context, value, encoded string) bool {
	if len(value) > 256 || !strings.HasPrefix(encoded, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(encoded, prefix), "$")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil || len(salt) != 16 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil || len(expected) != 32 {
		return false
	}
	actual, err := derive(ctx, value, salt)
	return err == nil && subtle.ConstantTimeCompare(expected, actual) == 1
}
