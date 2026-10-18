package sessions

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/websession"
)

var ErrMissing = errors.New("session unavailable")

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
type Service struct {
	db     *ent.Client
	key    []byte
	origin string
	secure bool
}

func New(db *ent.Client, encodedKey, origin string, secure bool) (*Service, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 || origin == "" {
		return nil, errors.New("invalid session configuration")
	}
	return &Service{db, key, origin, secure}, nil
}
func (s *Service) CookieName() string {
	if s.secure {
		return "__Host-moetopia_account"
	}
	return "moetopia_account"
}
func RandomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
func ValidToken(value string) bool {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(raw) == 32
}
func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func (s *Service) Cookie(token string) http.Cookie {
	return http.Cookie{Name: s.CookieName(), Value: token, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 7 * 24 * 60 * 60}
}
func (s *Service) CSRF(token string) string {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte("moetopia:account:csrf:v1:" + token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *Service) Protect(token, origin, csrf string) bool {
	return origin == s.origin && ValidToken(token) && subtle.ConstantTimeCompare([]byte(csrf), []byte(s.CSRF(token))) == 1
}
func (s *Service) Find(ctx context.Context, token string) (Session, error) {
	if !ValidToken(token) {
		return Session{}, ErrMissing
	}
	value, err := s.db.WebSession.Query().Where(websession.IDEQ(digest(token)), websession.ApplicationEQ("account"), websession.ExpiresAtGT(time.Now())).Only(ctx)
	if ent.IsNotFound(err) {
		return Session{}, ErrMissing
	}
	if err != nil {
		return Session{}, err
	}
	return Session{value.ID, value.UserID, value.CreatedAt, value.ExpiresAt}, nil
}
func (s *Service) Create(ctx context.Context, userID, previous string) (string, error) {
	token, err := RandomToken()
	if err != nil {
		return "", err
	}
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.WebSession.Delete().Where(websession.IDEQ(digest(previous)), websession.ApplicationEQ("account")).Exec(ctx); err != nil {
		return "", err
	}
	if err := tx.WebSession.Create().SetID(digest(token)).SetUserID(userID).SetApplication("account").SetExpiresAt(time.Now().Add(7 * 24 * time.Hour)).Exec(ctx); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return token, nil
}
func (s *Service) List(ctx context.Context, userID string) ([]Session, error) {
	values, err := s.db.WebSession.Query().Where(websession.UserIDEQ(userID), websession.ApplicationEQ("account"), websession.ExpiresAtGT(time.Now())).Order(ent.Desc(websession.FieldCreatedAt)).Limit(100).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Session, 0, len(values))
	for _, value := range values {
		result = append(result, Session{value.ID, value.UserID, value.CreatedAt, value.ExpiresAt})
	}
	return result, nil
}
func (s *Service) Revoke(ctx context.Context, userID, id string) error {
	_, err := s.db.WebSession.Delete().Where(websession.IDEQ(id), websession.UserIDEQ(userID), websession.ApplicationEQ("account")).Exec(ctx)
	return err
}
func (s *Service) Cleanup(ctx context.Context) error {
	_, err := s.db.WebSession.Delete().Where(websession.ExpiresAtLTE(time.Now())).Exec(ctx)
	return err
}
