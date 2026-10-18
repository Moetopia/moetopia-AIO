package main

import "testing"

func TestFixtureIsolation(t *testing.T) {
	name := "moetopia_test_0123456789abcdef"
	dsn := "postgres://test:test@localhost:55432/" + name
	callback := "http://localhost:54310/callback"
	if err := validate("test", dsn, name, callback); err != nil {
		t.Fatal(err)
	}
	for _, input := range []struct{ environment, dsn, name, callback string }{{"development", dsn, name, callback}, {"test", "postgres://test:test@localhost:55432/moetopia_dev", name, callback}, {"test", dsn, "moetopia_test_ffffffffffffffff", callback}, {"test", dsn, name, "https://outside.test/callback"}} {
		if validate(input.environment, input.dsn, input.name, input.callback) == nil {
			t.Fatal("unsafe OIDC fixture accepted")
		}
	}
}
