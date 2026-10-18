//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/valkey-io/valkey-go"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/sessions"
	"moetopia.local/aio/api/migrations"
)

type captureEmail struct{ values []notifications.Message }

func (c *captureEmail) Send(_ context.Context, message notifications.Message) error {
	c.values = append(c.values, message)
	return nil
}

type accountFixture struct {
	db       *persistence.Database
	identity *identity.Service
	sessions *sessions.Service
	outbox   *notifications.Outbox
	sender   *captureEmail
	handler  http.Handler
	ip       string
	cache    valkey.Client
}

func newAccountFixture(t *testing.T) *accountFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	u, err := url.Parse(dsn)
	if err != nil || !regexp.MustCompile(`^/moetopia_test_[a-f0-9]{16}$`).MatchString(u.Path) {
		t.Fatal("dedicated TEST_DATABASE_DSN is required")
	}
	db, err := persistence.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	provider, err := goose.NewProvider(goose.DialectPostgres, db.SQL, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(t.Context()); err != nil {
		t.Fatal(err)
	}
	cache, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{os.Getenv("TEST_VALKEY_ADDRESS")}, Password: os.Getenv("TEST_VALKEY_PASSWORD"), DisableCache: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cache.Close)
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	box, err := cryptobox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	sender := &captureEmail{}
	outbox := notifications.New(db.Ent, box, sender)
	accounts, err := identity.New(identity.NewStore(db.Ent, outbox), key)
	if err != nil {
		t.Fatal(err)
	}
	web, err := sessions.New(db.Ent, key, "https://account.moetopia.test", true)
	if err != nil {
		t.Fatal(err)
	}
	return &accountFixture{db, accounts, web, outbox, sender, New(Dependencies{Accounts: AccountEndpoints{Identity: accounts, Sessions: web, Cache: cache}}), "[2001:db8:" + uuid.NewString()[:4] + "::1]:1234", cache}
}
func (f *accountFixture) request(method, path, cookie, csrf, origin string, body any) *httptest.ResponseRecorder {
	encoded, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", origin)
	r.Header.Set("Cookie", cookie)
	r.Header.Set("X-Moetopia-CSRF", csrf)
	r.RemoteAddr = f.ip
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}
func decodeAccount[T any](t *testing.T, response *httptest.ResponseRecorder, status int) T {
	t.Helper()
	if response.Code != status {
		t.Fatalf("HTTP %d, expected %d", response.Code, status)
	}
	var result T
	if status != 204 {
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
	}
	return result
}
func cookieValue(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	values := w.Result().Cookies()
	if len(values) != 1 || !values[0].HttpOnly || !values[0].Secure || values[0].Domain != "" {
		t.Fatal("incorrect session cookie")
	}
	return values[0].Name + "=" + values[0].Value
}
func (f *accountFixture) verification(t *testing.T, email string) (string, string) {
	t.Helper()
	id, err := f.identity.Begin(t.Context(), email, "en")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.outbox.Deliver(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, message := range f.sender.values {
		if message.To == email {
			return id, regexp.MustCompile(`[0-9]{8}`).FindString(message.Text)
		}
	}
	t.Fatal("verification email missing")
	return "", ""
}

func TestAccountRegistrationLoginAndRevocation(t *testing.T) {
	f := newAccountFixture(t)
	ctx := t.Context()
	const origin = "https://account.moetopia.test"
	w := f.request("GET", "/api/v1/account/context", "", "", "", nil)
	anonymous := decodeAccount[AccountContextBody](t, w, 200)
	cookie := cookieValue(t, w)
	if anonymous.User != nil || anonymous.RegistrationMode != "invite" {
		t.Fatal("incorrect anonymous registration mode")
	}
	email := uuid.NewString() + "@account.test"
	loginBody := map[string]string{"email": email, "password": "a long test passphrase"}
	decodeAccount[any](t, f.request("POST", "/api/v1/account/login", cookie, anonymous.CSRFToken, "https://evil.test", loginBody), 403)
	decodeAccount[any](t, f.request("POST", "/api/v1/account/login", cookie, "", origin, loginBody), 403)
	challenge, code := f.verification(t, email)
	invite, err := f.identity.IssueInvitation(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]string{"challenge_id": challenge, "code": code, "invitation": "wrong", "display_name": "Test reader", "password": "a long test passphrase"}
	decodeAccount[any](t, f.request("POST", "/api/v1/account/register", cookie, anonymous.CSRFToken, origin, body), 422)
	body["invitation"] = invite
	decodeAccount[any](t, f.request("POST", "/api/v1/account/register", cookie, anonymous.CSRFToken, origin, body), 204)
	decodeAccount[any](t, f.request("POST", "/api/v1/account/register", cookie, anonymous.CSRFToken, origin, body), 422)
	loginBody["password"] = "incorrect password"
	decodeAccount[any](t, f.request("POST", "/api/v1/account/login", cookie, anonymous.CSRFToken, origin, loginBody), 401)
	loginBody["password"] = "a long test passphrase"
	w = f.request("POST", "/api/v1/account/login", cookie, anonymous.CSRFToken, origin, loginBody)
	authenticated := decodeAccount[AccountContextBody](t, w, 200)
	activeCookie := cookieValue(t, w)
	if authenticated.User == nil || authenticated.User.Email != email || activeCookie == cookie || authenticated.CSRFToken == anonymous.CSRFToken {
		t.Fatal("sign-in did not rotate the session")
	}
	old := decodeAccount[AccountContextBody](t, f.request("GET", "/api/v1/account/context", cookie, "", "", nil), 200)
	if old.User != nil {
		t.Fatal("anonymous cookie became authenticated")
	}
	list := decodeAccount[struct {
		Items     []sessions.Session `json:"items"`
		CurrentID string             `json:"current_id"`
	}](t, f.request("GET", "/api/v1/account/sessions", activeCookie, "", "", nil), 200)
	if len(list.Items) != 1 || list.CurrentID == "" {
		t.Fatal("session list incorrect")
	}
	otherToken, err := f.sessions.Create(ctx, uuid.NewString(), "")
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.sessions.Find(ctx, otherToken)
	if err != nil {
		t.Fatal(err)
	}
	decodeAccount[any](t, f.request("DELETE", "/api/v1/account/sessions/"+other.ID, activeCookie, authenticated.CSRFToken, origin, nil), 204)
	if _, err := f.sessions.Find(ctx, otherToken); err != nil {
		t.Fatal("another account's session was revoked")
	}
	decodeAccount[any](t, f.request("POST", "/api/v1/account/logout", activeCookie, authenticated.CSRFToken, origin, nil), 200)
	if _, err := f.sessions.Find(ctx, w.Result().Cookies()[0].Value); !errors.Is(err, sessions.ErrMissing) {
		t.Fatal("signed-out session remained valid")
	}
	decodeAccount[any](t, f.request("GET", "/api/v1/account/sessions", activeCookie, "", "", nil), 401)
}

func TestInvitationAtomicityAndVerificationAttempts(t *testing.T) {
	f := newAccountFixture(t)
	invite, err := f.identity.IssueInvitation(t.Context(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	id1, code1 := f.verification(t, uuid.NewString()+"@concurrency.test")
	id2, code2 := f.verification(t, uuid.NewString()+"@concurrency.test")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, input := range []struct{ id, code string }{{id1, code1}, {id2, code2}} {
		wg.Go(func() {
			results <- f.identity.Register(t.Context(), input.id, input.code, invite, "Concurrent reader", "another long test passphrase")
		})
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, identity.ErrInvalid) {
			t.Fatal("unexpected registration failure")
		}
	}
	if successes != 1 {
		t.Fatal("single-use invitation consumed more or less than once")
	}
	id3, code3 := f.verification(t, uuid.NewString()+"@attempts.test")
	invite, err = f.identity.IssueInvitation(t.Context(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if err := f.identity.Register(t.Context(), id3, "not-code", invite, "Reader", "another long test passphrase"); !errors.Is(err, identity.ErrInvalid) {
			t.Fatal("incorrect code accepted")
		}
	}
	if err := f.identity.Register(t.Context(), id3, code3, invite, "Reader", "another long test passphrase"); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("exhausted challenge remained usable")
	}
}
