//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/identity"
)

func TestAdminUserAvailabilityAndProtectedAdministrators(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	other, portalCookie, _ := managementFixture(t)
	target, err := other.identity.User(t.Context(), other.userID)
	if err != nil {
		t.Fatal(err)
	}
	search := f.request("GET", "/api/v1/admin/users?query="+other.userID, cookie, "", "", nil)
	items := decodeAccount[identity.ManagedUsers](t, search, 200)
	if len(items.Items) != 1 || items.Items[0].ID != other.userID || bytes.Contains(search.Body.Bytes(), []byte("password_hash")) {
		t.Fatal("admin user search leaked private credentials or returned an incorrect account")
	}
	if got := f.request("POST", "/api/v1/admin/users/"+f.userID+"/availability", cookie, csrf, adminOrigin, map[string]any{"revision": 0, "enabled": false, "reason": "Attempted self suspension"}).Code; got != 403 {
		t.Fatal("ordinary account controls can disable an administrator")
	}
	result := decodeAccount[identity.ManagedUser](t, f.request("POST", "/api/v1/admin/users/"+other.userID+"/availability", cookie, csrf, adminOrigin, map[string]any{"revision": target.Revision, "enabled": false, "reason": "Account abuse investigation"}), 200)
	if result.Enabled || result.DisabledReason == "" {
		t.Fatal("account was not disabled")
	}
	if got := other.request("GET", "/api/v1/portal/sessions", portalCookie, "", "", nil).Code; got != 401 {
		t.Fatal("disabled user retains RP access")
	}
	if _, err := other.identity.Login(t.Context(), target.Email, initialPassword); !errors.Is(err, identity.ErrCredentials) {
		t.Fatal("disabled account can log in")
	}
	decodeAccount[identity.ManagedUser](t, f.request("POST", "/api/v1/admin/users/"+other.userID+"/availability", cookie, csrf, adminOrigin, map[string]any{"revision": result.Revision, "enabled": true, "reason": ""}), 200)
	if _, err := other.identity.Login(t.Context(), target.Email, initialPassword); err != nil {
		t.Fatal("reenabled account cannot log in with its credential")
	}
	if got := other.request("GET", "/api/v1/portal/sessions", portalCookie, "", "", nil).Code; got != 401 {
		t.Fatal("reenabling restored an old session")
	}
	events := decodeAccount[audit.AdminEvents](t, f.request("GET", "/api/v1/admin/audit-events", cookie, "", "", nil), 200)
	found := false
	for _, event := range events.Items {
		if event.ActorID == f.userID && event.TargetID == other.userID && event.Action == "account.disabled" && event.RequestID != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("account administration audit unavailable")
	}
}
func TestAdminInvitationsAndRegistrationPolicy(t *testing.T) {
	f := newAdminFixture(t)
	cookie, csrf := f.login(t)
	cookie, csrf, _, _ = f.enroll(t, cookie, csrf)
	policy := decodeAccount[identity.RegistrationPolicy](t, f.request("GET", "/api/v1/admin/registration-policy", cookie, "", "", nil), 200)
	t.Cleanup(func() {
		current, err := f.identity.RegistrationPolicy(context.Background())
		if err != nil {
			t.Error(err)
			return
		}
		if _, err := f.identity.SetRegistrationPolicy(context.Background(), f.userID, policy.Mode, current.Revision); err != nil {
			t.Error(err)
		}
	})
	open := decodeAccount[identity.RegistrationPolicy](t, f.request("PUT", "/api/v1/admin/registration-policy", cookie, csrf, adminOrigin, map[string]any{"mode": "open", "revision": policy.Revision}), 200)
	if open.Mode != "open" || open.Revision != policy.Revision+1 {
		t.Fatal("registration policy did not change")
	}
	if got := f.request("PUT", "/api/v1/admin/registration-policy", cookie, csrf, adminOrigin, map[string]any{"mode": "invite", "revision": policy.Revision}).Code; got != 409 {
		t.Fatal("stale policy overwrote newer state")
	}
	decodeAccount[identity.RegistrationPolicy](t, f.request("PUT", "/api/v1/admin/registration-policy", cookie, csrf, adminOrigin, map[string]any{"mode": "invite", "revision": open.Revision}), 200)
	issued := decodeAccount[identity.IssuedInvitation](t, f.request("POST", "/api/v1/admin/invitations", cookie, csrf, adminOrigin, map[string]int{"expires_in_hours": 24}), 200)
	if issued.Code == "" || issued.ID == issued.Code {
		t.Fatal("invitation identifier must not be its bearer secret")
	}
	list := f.request("GET", "/api/v1/admin/invitations", cookie, "", "", nil)
	if list.Code != 200 || bytes.Contains(list.Body.Bytes(), []byte(issued.Code)) {
		t.Fatal("invitation list disclosed bearer code")
	}
	if got := f.request("DELETE", "/api/v1/admin/invitations/"+issued.ID, cookie, csrf, adminOrigin, nil).Code; got != 204 {
		t.Fatal("invitation revocation failed")
	}
	challenge, code := f.verification(t, uuid.NewString()+"@example.test")
	if err := f.identity.Register(t.Context(), challenge, code, issued.Code, "Revoked invite fixture", initialPassword); !errors.Is(err, identity.ErrInvalid) {
		t.Fatal("revoked invitation can still create an account")
	}
}
