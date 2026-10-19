package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/password"
)

var ErrInvalid = errors.New("invalid or expired registration details")
var ErrCredentials = errors.New("invalid credentials")
var ErrMissing = errors.New("account not available")

type User struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	DisplayName     string `json:"display_name"`
	Bio             string `json:"bio" required:"false"`
	AvatarURL       string `json:"avatar_url" required:"false"`
	Revision        int    `json:"revision" required:"false"`
	SecurityVersion int    `json:"-"`
}
type Service struct {
	store  *Store
	secret []byte
	dummy  string
}

func New(store *Store, encodedKey string) (*Service, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("invalid identity secret")
	}
	dummy, err := password.Hash(context.Background(), uuid.NewString())
	if err != nil {
		return nil, err
	}
	return &Service{store, key, dummy}, nil
}

func NormalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || len(value) > 254 || !strings.Contains(value, ".") || strings.ContainsAny(value, "\r\n") {
		return "", ErrInvalid
	}
	for _, r := range value {
		if r > 127 {
			return "", ErrInvalid
		}
	}
	return value, nil
}

func (s *Service) codeHash(id, email, code string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = fmt.Fprintf(mac, "moetopia:registration:v1:%s:%s:%s", id, email, code)
	return mac.Sum(nil)
}

func (s *Service) Begin(ctx context.Context, inputEmail, locale string) (string, error) {
	email, err := NormalizeEmail(inputEmail)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	exists, err := s.store.Exists(ctx, email)
	if err != nil {
		return "", err
	}
	if exists {
		return id, nil
	}
	number, err := rand.Int(rand.Reader, big.NewInt(100000000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%08d", number.Int64())
	err = s.store.Begin(ctx, id, email, s.codeHash(id, email, code), code, locale, time.Now().Add(10*time.Minute))
	return id, err
}

func (s *Service) Register(ctx context.Context, id, code, invite, name, value string) error {
	name = strings.TrimSpace(name)
	if len(code) != 8 || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 80 || !password.Valid(value) {
		return ErrInvalid
	}
	verification, err := s.store.Verification(ctx, id)
	if err != nil {
		return err
	}
	if verification.Consumed || verification.Attempts >= 5 || !verification.ExpiresAt.After(time.Now()) {
		return ErrInvalid
	}
	if subtle.ConstantTimeCompare(verification.Hash, s.codeHash(id, verification.Email, code)) != 1 {
		if err := s.store.FailedAttempt(ctx, id); err != nil {
			return err
		}
		return ErrInvalid
	}
	hash, err := password.Hash(ctx, value)
	if err != nil {
		return err
	}
	return s.store.Register(ctx, id, verification.Email, name, hash, invitationHash(strings.TrimSpace(invite)))
}

func (s *Service) Login(ctx context.Context, email, value string) (User, error) {
	email, err := NormalizeEmail(email)
	if err != nil {
		email = "invalid@example.invalid"
	}
	user, hash, err := s.store.Credentials(ctx, email)
	if err != nil && !errors.Is(err, ErrMissing) {
		return User{}, err
	}
	if errors.Is(err, ErrMissing) {
		hash = s.dummy
	}
	valid := password.Verify(ctx, value, hash)
	if !valid || err != nil {
		return User{}, ErrCredentials
	}
	return user, nil
}

func (s *Service) User(ctx context.Context, id string) (User, error) { return s.store.User(ctx, id) }
func (s *Service) Mode(ctx context.Context) (string, error)          { return s.store.Mode(ctx) }
func (s *Service) Cleanup(ctx context.Context) error                 { return s.store.Cleanup(ctx) }

func invitationHash(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
func (s *Service) IssueInvitation(ctx context.Context, expires time.Time) (string, error) {
	code := make([]byte, 24)
	if _, err := rand.Read(code); err != nil {
		return "", err
	}
	plain := hex.EncodeToString(code)
	return plain, s.store.IssueInvitation(ctx, invitationHash(plain), expires)
}
