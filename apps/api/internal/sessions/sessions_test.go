package sessions

import "testing"

func TestCSRFAndCookieBoundary(t *testing.T) {
	service, err := New(nil, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", "https://account.example.test", true)
	if err != nil {
		t.Fatal(err)
	}
	token, err := RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	csrf := service.CSRF(token)
	if !service.Protect(token, "https://account.example.test", csrf) {
		t.Fatal("valid request rejected")
	}
	for _, input := range []struct{ token, origin, csrf string }{
		{token, "https://evil.example.test", csrf}, {token, "", csrf}, {token, "https://account.example.test", "wrong"}, {"forged", "https://account.example.test", csrf},
	} {
		if service.Protect(input.token, input.origin, input.csrf) {
			t.Fatal("forged write accepted")
		}
	}
	cookie := service.Cookie(token)
	if !cookie.HttpOnly || !cookie.Secure || cookie.Domain != "" || cookie.Path != "/" || cookie.Name != "__Host-moetopia_account" {
		t.Fatal("cookie protection changed")
	}
}
