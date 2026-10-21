//go:build integration

package federation

import (
	"context"
	"encoding/base64"
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
	"moetopia.local/aio/api/internal/persistence/ent/emailverification"
	"moetopia.local/aio/api/internal/persistence/ent/externalidentity"
	"moetopia.local/aio/api/internal/persistence/ent/externallogin"
	"moetopia.local/aio/api/internal/sessions"
	"moetopia.local/aio/api/migrations"
)

type fakeUpstream struct {
	profile    Profile
	calls      int
	onExchange func()
}

func (p *fakeUpstream) AuthorizationURL(state, nonce, verifier string) string {
	return "https://github.com/login/oauth/authorize?state=" + state
}
func (p *fakeUpstream) Exchange(context.Context, string, string, string) (Profile, error) {
	p.calls++
	if p.onExchange != nil {
		p.onExchange()
	}
	return p.profile, nil
}

type emailCapture struct{ messages []notifications.Message }

func (s *emailCapture) Send(_ context.Context, message notifications.Message) error {
	s.messages = append(s.messages, message)
	return nil
}

type flowFixture struct {
	db       *persistence.Database
	service  *Service
	users    *identity.Service
	web      *sessions.Service
	outbox   *notifications.Outbox
	email    *emailCapture
	provider *fakeUpstream
}

const testPassword = "long federation test password 2026"

func newFlowFixture(t *testing.T) *flowFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	u, err := url.Parse(dsn)
	if err != nil || !regexp.MustCompile(`^/moetopia_test_[a-f0-9]{16}$`).MatchString(u.Path) {
		t.Fatal("dedicated TEST_DATABASE_DSN required")
	}
	db, err := persistence.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migrator, err := goose.NewProvider(goose.DialectPostgres, db.SQL, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrator.Up(t.Context()); err != nil {
		t.Fatal(err)
	}
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	box, err := cryptobox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	web, err := sessions.NewPortal(db.Ent, key, "https://account.moetopia.test", true)
	if err != nil {
		t.Fatal(err)
	}
	email := &emailCapture{}
	outbox := notifications.New(db.Ent, box, email)
	users, err := identity.New(identity.NewStore(db.Ent, outbox).WithSessions(web), key)
	if err != nil {
		t.Fatal(err)
	}
	provider := &fakeUpstream{profile: Profile{Provider: "github", Issuer: "https://github.com", Subject: uuid.NewString(), Email: "hint@example.test", DisplayName: "Provider hint"}}
	service := NewService(db.Ent, box, users, nil, web)
	service.providers["github"] = provider
	return &flowFixture{db, service, users, web, outbox, email, provider}
}
func (f *flowFixture) begin(t *testing.T, binding *Binding) (Started, string) {
	t.Helper()
	start, err := f.service.Begin(t.Context(), "github", "", Continuation{Developer: true}, binding)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(start.RedirectURL)
	if err != nil {
		t.Fatal(err)
	}
	return start, u.Query().Get("state")
}
func (f *flowFixture) registration(t *testing.T) (string, string, string, string) {
	t.Helper()
	email := uuid.NewString() + "@example.test"
	id, err := f.users.Begin(t.Context(), email, "en")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.outbox.Deliver(t.Context()); err != nil {
		t.Fatal(err)
	}
	code := ""
	for _, message := range f.email.messages {
		if message.To == email {
			code = regexp.MustCompile(`\b[0-9]{8}\b`).FindString(message.Text)
		}
	}
	if code == "" {
		t.Fatal("missing registration code")
	}
	invite, err := f.users.IssueInvitation(t.Context(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return email, id, code, invite
}
func (f *flowFixture) ordinary(t *testing.T) identity.User {
	t.Helper()
	email, id, code, invite := f.registration(t)
	if err := f.users.Register(t.Context(), id, code, invite, "Ordinary", testPassword); err != nil {
		t.Fatal(err)
	}
	user, err := f.users.Login(t.Context(), email, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func TestExternalRegistrationPolicyAndReplay(t *testing.T) {
	f := newFlowFixture(t)
	start, state := f.begin(t, nil)
	otherBrowser, _ := sessions.RandomToken()
	if _, err := f.service.Callback(t.Context(), "github", state, otherBrowser, "code", false, nil); !errors.Is(err, ErrFlow) {
		t.Fatal("callback ignored browser binding")
	}
	result, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, nil)
	if err != nil || result.Completion == "" || result.User != nil {
		t.Fatalf("missing first-login registration: %v", err)
	}
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, nil); !errors.Is(err, ErrFlow) || f.provider.calls != 1 {
		t.Fatal("authorization state replayed")
	}
	if _, err := f.service.Registration(t.Context(), result.Completion, otherBrowser); !errors.Is(err, ErrFlow) {
		t.Fatal("registration proof crossed browsers")
	}
	details, err := f.service.Registration(t.Context(), result.Completion, start.Browser)
	if err != nil || details.EmailHint != "hint@example.test" {
		t.Fatal("missing safe registration hint")
	}
	email, challenge, code, invite := f.registration(t)
	if _, err := f.service.Complete(t.Context(), result.Completion, start.Browser, challenge, "bad-code", invite, "Local name"); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("registration skipped local email proof")
	}
	verification, err := f.db.Ent.EmailVerification.Get(t.Context(), challenge)
	if err != nil || verification.Attempts != 1 {
		t.Fatal("invalid code attempt was not persisted")
	}
	if _, err := f.service.Complete(t.Context(), result.Completion, start.Browser, challenge, code, "invalid-invite", "Local name"); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("registration bypassed invitation policy")
	}
	f.db.SQL.SetMaxOpenConns(1)
	registered, err := f.service.Complete(t.Context(), result.Completion, start.Browser, challenge, code, invite, "Local name")
	if err != nil {
		t.Fatal(err)
	}
	if registered.User.HasPassword || registered.User.Email != email || registered.User.DisplayName != "Local name" || !registered.Continuation.Developer {
		t.Fatal("external account incorrectly inherited provider email or password")
	}
	if _, err := f.users.Login(t.Context(), email, "!external-only"); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("passwordless account has a usable local password")
	}
	if _, err := f.service.Complete(t.Context(), result.Completion, start.Browser, challenge, code, invite, "Local name"); !errors.Is(err, ErrFlow) {
		t.Fatal("registration proof replayed")
	}
	start, state = f.begin(t, nil)
	login, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, nil)
	if err != nil || login.User == nil || login.User.ID != registered.User.ID {
		t.Fatalf("bound identity did not sign in: %v", err)
	}
	if _, err := f.web.CreateVerified(t.Context(), login.User.ID, "", login.User.SecurityVersion, f.users); err != nil {
		t.Fatal(err)
	}
	if err := f.users.UnbindExternal(t.Context(), *registered.User, "github", testPassword); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("passwordless identity could remove its login method")
	}
	resetID, err := f.users.BeginReset(t.Context(), email, "en")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.outbox.Deliver(t.Context()); err != nil {
		t.Fatal(err)
	}
	resetCode := ""
	for _, message := range f.email.messages {
		if message.To == email && strings.Contains(message.Text, resetID) {
			resetCode = regexp.MustCompile(`\b[0-9]{8}\b`).FindString(message.Text)
		}
	}
	if err := f.users.ResetPassword(t.Context(), resetID, resetCode, testPassword); err != nil {
		t.Fatal(err)
	}
	user, err := f.users.Login(t.Context(), email, testPassword)
	if err != nil || !user.HasPassword {
		t.Fatal("verified recovery did not establish a local password")
	}
	if err := f.users.UnbindExternal(t.Context(), user, "github", testPassword); err != nil {
		t.Fatal(err)
	}
}

func TestExternalBindingSessionAndTransactionSafety(t *testing.T) {
	f := newFlowFixture(t)
	user := f.ordinary(t)
	token, err := f.web.Create(t.Context(), user.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	web, err := f.web.Find(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	binding := &Binding{User: user, SessionID: web.ID}
	start, state := f.begin(t, binding)
	other := &Binding{User: user, SessionID: "different-session"}
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, other); !errors.Is(err, ErrFlow) {
		t.Fatal("binding accepted a switched session")
	}
	start, state = f.begin(t, binding)
	failNotice := true
	f.db.Ent.EmailDelivery.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			if failNotice {
				return nil, errors.New("injected outbox failure")
			}
			return next.Mutate(ctx, mutation)
		})
	})
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, binding); err == nil {
		t.Fatal("binding committed without security notice")
	}
	failNotice = false
	if exists, err := f.db.Ent.ExternalIdentity.Query().Where(externalidentity.UserID(user.ID)).Exist(t.Context()); err != nil || exists {
		t.Fatal("failed binding left an external credential")
	}
	if _, err := f.web.Find(t.Context(), token); err != nil {
		t.Fatal("failed binding revoked valid session")
	}
	start, state = f.begin(t, binding)
	result, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, binding)
	if err != nil || !result.Bound {
		t.Fatalf("binding failed: %v", err)
	}
	if _, err := f.web.Find(t.Context(), token); !errors.Is(err, sessions.ErrMissing) {
		t.Fatal("credential binding retained old session")
	}
	updated, err := f.users.User(t.Context(), user.ID)
	if err != nil || updated.SecurityVersion != user.SecurityVersion+1 {
		t.Fatal("binding did not update security version")
	}
	bindings, err := f.users.ExternalBindings(t.Context(), user.ID)
	if err != nil || len(bindings) != 1 || bindings[0].Provider != "github" {
		t.Fatal("missing external binding summary")
	}
	otherUser := f.ordinary(t)
	if err := f.users.BindExternal(t.Context(), otherUser, identity.External{Provider: "github", Issuer: "https://github.com", Subject: f.provider.profile.Subject}); !errors.Is(err, identity.ErrExternalConflict) {
		t.Fatal("external identity linked to two users")
	}
	start, state = f.begin(t, nil)
	if err := f.db.Ent.AccountUser.UpdateOneID(user.ID).SetEnabled(false).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, nil); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("disabled external user could sign in or register again")
	}
	if err := f.db.Ent.AccountUser.UpdateOneID(user.ID).SetEnabled(true).SetDeletionRequestedAt(time.Now().Add(-8 * 24 * time.Hour)).SetDeletionDueAt(time.Now().Add(-time.Hour)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	start, state = f.begin(t, nil)
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, nil); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("pending external user could sign in")
	}
	if err := f.users.PurgeDeletion(t.Context(), user.ID); err != nil {
		t.Fatal(err)
	}
	if exists, err := f.db.Ent.ExternalIdentity.Query().Where(externalidentity.UserID(user.ID)).Exist(t.Context()); err != nil || exists {
		t.Fatal("deletion retained external identity")
	}
	if exists, err := f.db.Ent.ExternalLogin.Query().Where(externallogin.UserID(user.ID)).Exist(t.Context()); err != nil || exists {
		t.Fatal("deletion retained binding transactions")
	}
	if _, err := f.users.User(t.Context(), otherUser.ID); err != nil {
		t.Fatal("deletion affected another account")
	}
}

func TestExternalRegistrationDoesNotMergeExistingEmail(t *testing.T) {
	f := newFlowFixture(t)
	email, challenge, code, invite := f.registration(t)
	second, err := f.users.Begin(t.Context(), email, "en")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.outbox.Deliver(t.Context()); err != nil {
		t.Fatal(err)
	}
	secondCode := ""
	for _, message := range f.email.messages {
		if message.To == email {
			secondCode = regexp.MustCompile(`\b[0-9]{8}\b`).FindString(message.Text)
		}
	}
	if err := f.users.Register(t.Context(), challenge, code, invite, "Existing", testPassword); err != nil {
		t.Fatal(err)
	}
	f.provider.profile.Email = email
	start, state := f.begin(t, nil)
	result, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, nil)
	if err != nil || result.User != nil || result.Completion == "" {
		t.Fatal("provider email silently matched a local account")
	}
	secondInvite, err := f.users.IssueInvitation(t.Context(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Complete(t.Context(), result.Completion, start.Browser, second, secondCode, secondInvite, "Attempted merge"); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("duplicate local email did not require explicit linking")
	}
	if exists, err := f.db.Ent.ExternalIdentity.Query().Where(externalidentity.Subject(f.provider.profile.Subject)).Exist(t.Context()); err != nil || exists {
		t.Fatal("failed registration attached a credential")
	}
	row, err := f.db.Ent.EmailVerification.Get(t.Context(), second)
	if err != nil || row.Consumed {
		t.Fatal("failed registration consumed email proof")
	}
	if _, err := f.service.Registration(t.Context(), result.Completion, start.Browser); err != nil {
		t.Fatal("failed registration consumed completion proof")
	}
	if count, err := f.db.Ent.AccountUser.Query().Where(accountuser.Email(email)).Count(t.Context()); err != nil || count != 1 {
		t.Fatal("duplicate account created")
	}
	if _, err := f.db.Ent.EmailVerification.Delete().Where(emailverification.ID(second)).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestExternalStateIsolationAndLateSessionRevocation(t *testing.T) {
	f := newFlowFixture(t)
	user := f.ordinary(t)
	token, err := f.web.Create(t.Context(), user.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	web, err := f.web.Find(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	binding := &Binding{User: user, SessionID: web.ID}
	start, state := f.begin(t, binding)
	f.service.providers["google"] = f.provider
	if _, err := f.service.Callback(t.Context(), "google", state, start.Browser, "code", false, binding); !errors.Is(err, ErrFlow) || f.provider.calls != 0 {
		t.Fatal("state crossed providers")
	}
	f.provider.onExchange = func() {
		if err := f.web.Revoke(t.Context(), user.ID, web.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, binding); !errors.Is(err, sessions.ErrMissing) {
		t.Fatal("binding survived session revocation during provider exchange")
	}
	if exists, err := f.db.Ent.ExternalIdentity.Query().Where(externalidentity.UserID(user.ID)).Exist(t.Context()); err != nil || exists {
		t.Fatal("late revoked session attached identity")
	}
	f.provider.onExchange = nil
	start, state = f.begin(t, nil)
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "", true, nil); !errors.Is(err, ErrProvider) {
		t.Fatal("denied authorization accepted")
	}
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, nil); !errors.Is(err, ErrFlow) {
		t.Fatal("denied authorization could be replayed")
	}
	start, state = f.begin(t, nil)
	if _, err := f.db.SQL.ExecContext(t.Context(), "UPDATE identity.external_logins SET expires_at=$1 WHERE id=$2", time.Now().Add(-time.Minute), digest("external-state:"+state)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Callback(t.Context(), "github", state, start.Browser, "code", false, nil); !errors.Is(err, ErrFlow) {
		t.Fatal("expired authorization accepted")
	}
	if err := f.users.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	if exists, err := f.db.Ent.ExternalLogin.Query().Where(externallogin.ID(digest("external-state:" + state))).Exist(t.Context()); err != nil || exists {
		t.Fatal("expired authorization not removed")
	}
}
