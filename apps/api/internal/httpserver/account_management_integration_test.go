//go:build integration

package httpserver

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math/rand/v2"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/password"
)

const initialPassword = "initial account test passphrase"
const nextPassword = "updated account test passphrase"

func managementFixture(t *testing.T) (*portalFixture, string, string) {
	t.Helper()
	f := newPortalFixture(t)
	hash, err := password.Hash(t.Context(), initialPassword)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Ent.AccountUser.UpdateOneID(f.userID).SetPasswordHash(hash).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	token, err := f.portal.Sessions.CreateVerified(t.Context(), f.userID, "", 0, f.identity)
	if err != nil {
		t.Fatal(err)
	}
	return f, f.portal.Sessions.CookieName() + "=" + token, f.portal.Sessions.CSRF(token)
}
func resetCode(t *testing.T, f *portalFixture, email string) string {
	t.Helper()
	for range 10 {
		if err := f.outbox.Deliver(t.Context()); err != nil {
			t.Fatal(err)
		}
		for i := len(f.sender.values) - 1; i >= 0; i-- {
			message := f.sender.values[i]
			if message.To == email {
				if code := regexp.MustCompile(`[0-9]{8}`).FindString(message.Text); code != "" {
					return code
				}
			}
		}
	}
	t.Fatal("verification message missing")
	return ""
}
func currentEmail(t *testing.T, f *portalFixture) string {
	t.Helper()
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	return user.Email
}

func TestAccountManagementProfileAvatarAndAudit(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	user := decodeAccount[identity.User](t, f.request("PATCH", "/api/v1/portal/profile", cookie, csrf, portalOrigin, map[string]any{"display_name": "Updated reader", "bio": "A small biography", "revision": 0}), 200)
	if user.Revision != 1 || user.Bio != "A small biography" || user.DisplayName != "Updated reader" {
		t.Fatal("profile update missing")
	}
	if got := f.request("PATCH", "/api/v1/portal/profile", cookie, csrf, portalOrigin, map[string]any{"display_name": "Lost update", "bio": "", "revision": 0}).Code; got != 409 {
		t.Fatal("stale profile revision accepted")
	}
	if got := f.request("PATCH", "/api/v1/portal/profile", f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]any{"display_name": "Wrong session", "bio": "", "revision": 1}).Code; got != 403 {
		t.Fatal("identity session can edit RP profile")
	}
	source := image.NewNRGBA(image.Rect(0, 0, 400, 200))
	noise := rand.New(rand.NewPCG(42, 24))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			value := noise.Uint32()
			source.Set(x, y, color.NRGBA{R: uint8(value), G: uint8(value >> 8), B: uint8(value >> 16), A: 255})
		}
	}
	var data bytes.Buffer
	if err := png.Encode(&data, source); err != nil {
		t.Fatal(err)
	}
	if data.Len() < 64<<10 {
		t.Fatal("avatar fixture must exercise the larger upload route limit")
	}
	user = decodeAccount[identity.User](t, f.request("PUT", "/api/v1/portal/avatar", cookie, csrf, portalOrigin, map[string]any{"image_base64": base64.StdEncoding.EncodeToString(data.Bytes()), "revision": 1}), 200)
	if user.AvatarURL == "" || user.Revision != 2 {
		t.Fatal("avatar metadata missing")
	}
	avatar := f.request("GET", user.AvatarURL, "", "", "", nil)
	decoded, err := png.Decode(bytes.NewReader(avatar.Body.Bytes()))
	if avatar.Code != 200 || err != nil || decoded.Bounds().Dx() != 256 || decoded.Bounds().Dy() != 256 {
		t.Fatal("avatar was not normalized PNG")
	}
	if got := f.request("PUT", "/api/v1/portal/avatar", cookie, csrf, portalOrigin, map[string]any{"image_base64": base64.StdEncoding.EncodeToString([]byte("<svg onload=alert(1) />")), "revision": 2}).Code; got != 422 {
		t.Fatal("active image content accepted")
	}
	user = decodeAccount[identity.User](t, f.request("PUT", "/api/v1/portal/avatar", cookie, csrf, portalOrigin, map[string]any{"image_base64": "", "revision": 2}), 200)
	if user.AvatarURL != "" || f.request("GET", "/api/v1/avatars/"+f.userID, "", "", "", nil).Code != 404 {
		t.Fatal("avatar removal failed")
	}
	events := decodeAccount[struct {
		Items []struct {
			Action string `json:"action"`
		} `json:"items"`
	}](t, f.request("GET", "/api/v1/portal/security-events", cookie, "", "", nil), 200)
	if len(events.Items) != 3 {
		t.Fatal("profile audit not atomic or missing")
	}
}

func TestAccountManagementPasswordInvalidation(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	if got := f.request("POST", "/api/v1/portal/password", cookie, csrf, portalOrigin, map[string]string{"current_password": "wrong credential", "new_password": nextPassword}).Code; got != 401 {
		t.Fatal("current password not required")
	}
	if got := f.request("POST", "/api/v1/portal/password", cookie, csrf, portalOrigin, map[string]string{"current_password": initialPassword, "new_password": nextPassword}).Code; got != 204 {
		t.Fatalf("password change HTTP %d", got)
	}
	if _, err := f.sessions.Find(t.Context(), f.identityToken); err == nil {
		t.Fatal("identity session survived password change")
	}
	if got := f.request("GET", "/api/v1/portal/sessions", cookie, "", "", nil).Code; got != 401 {
		t.Fatal("RP session survived password change")
	}
	email := currentEmail(t, f)
	if _, err := f.identity.Login(t.Context(), email, initialPassword); err == nil {
		t.Fatal("old password remains valid")
	}
	user, err := f.identity.Login(t.Context(), email, nextPassword)
	if err != nil || user.SecurityVersion != 1 {
		t.Fatal("new password or security version missing")
	}
	if _, err := f.portal.Sessions.CreateVerified(t.Context(), f.userID, "", 0, f.identity); err == nil {
		t.Fatal("stale authentication result can recreate a session")
	}
}

func TestAccountManagementRecoveryAndSingleUse(t *testing.T) {
	f, _, _ := managementFixture(t)
	email := currentEmail(t, f)
	begin := func(email string) string {
		result := decodeAccount[struct {
			ChallengeID string `json:"challenge_id"`
		}](t, f.request("POST", "/api/v1/account/password-resets", f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"email": email, "locale": "en"}), 200)
		if _, err := uuid.Parse(result.ChallengeID); err != nil {
			t.Fatal("recovery response leaked account existence")
		}
		return result.ChallengeID
	}
	missing := uuid.NewString() + "@example.test"
	_ = begin(missing)
	id := begin(email)
	code := resetCode(t, f, email)
	for _, message := range f.sender.values {
		if message.To == missing {
			t.Fatal("unknown-account recovery sent mail")
		}
	}
	wrong := "00000000"
	if code == wrong {
		wrong = "11111111"
	}
	for range 5 {
		if got := f.request("POST", "/api/v1/account/password-resets/confirm", f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"challenge_id": id, "code": wrong, "new_password": nextPassword}).Code; got != 422 {
			t.Fatal("wrong recovery code accepted")
		}
	}
	if got := f.request("POST", "/api/v1/account/password-resets/confirm", f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"challenge_id": id, "code": code, "new_password": nextPassword}).Code; got != 422 {
		t.Fatal("exhausted code accepted")
	}
	id = begin(email)
	code = resetCode(t, f, email)
	var group sync.WaitGroup
	statuses := make(chan int, 2)
	for range 2 {
		group.Go(func() {
			statuses <- f.request("POST", "/api/v1/account/password-resets/confirm", f.identityCookie, f.sessions.CSRF(f.identityToken), portalOrigin, map[string]string{"challenge_id": id, "code": code, "new_password": nextPassword}).Code
		})
	}
	group.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[204] != 1 || counts[422] != 1 {
		t.Fatal("recovery transaction was not single-use")
	}
	if _, err := f.identity.Login(t.Context(), email, nextPassword); err != nil {
		t.Fatal("recovered password rejected")
	}
	if _, err := f.sessions.Find(t.Context(), f.identityToken); err == nil {
		t.Fatal("recovery preserved old identity session")
	}
}

func TestAccountManagementEmailChangeBinding(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	old := currentEmail(t, f)
	target := uuid.NewString() + "@example.test"
	response := decodeAccount[struct {
		ChallengeID string `json:"challenge_id"`
	}](t, f.request("POST", "/api/v1/portal/email-changes", cookie, csrf, portalOrigin, map[string]string{"current_password": initialPassword, "email": target, "locale": "en"}), 200)
	code := resetCode(t, f, target)
	if err := f.identity.ResetPassword(t.Context(), response.ChallengeID, code, nextPassword); err == nil {
		t.Fatal("email challenge can reset passwords")
	}
	if err := f.identity.ConfirmEmailChange(t.Context(), f.userID, "wrong-session", response.ChallengeID, code, initialPassword); err == nil {
		t.Fatal("email change browser binding missing")
	}
	if got := f.request("POST", "/api/v1/portal/email-changes/confirm", cookie, csrf, portalOrigin, map[string]string{"current_password": initialPassword, "challenge_id": response.ChallengeID, "code": code}).Code; got != 204 {
		t.Fatalf("email confirmation HTTP %d", got)
	}
	if _, err := f.identity.Login(t.Context(), old, initialPassword); err == nil {
		t.Fatal("old email can still log in")
	}
	if _, err := f.identity.Login(t.Context(), target, initialPassword); err != nil {
		t.Fatal("verified new email rejected")
	}
	if got := f.request("GET", "/api/v1/portal/sessions", cookie, "", "", nil).Code; got != 401 {
		t.Fatal("email change retained RP sessions")
	}
	if err := f.outbox.Deliver(t.Context()); err != nil {
		t.Fatal(err)
	}
	notified := false
	for _, message := range f.sender.values {
		if message.To == old && strings.Contains(message.Text, "email.changed") {
			notified = true
		}
	}
	if !notified {
		t.Fatal("old email not notified")
	}
}
