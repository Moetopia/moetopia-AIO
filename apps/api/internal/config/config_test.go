package config

import (
	"strings"
	"testing"
)

func TestRejectUnsafeIssuerWithoutExposingSecrets(t *testing.T) {
	for _, issuer := range []string{"http://example.com", "https://user:secret@example.com", "https://example.com/?secret=secret", "https://example.com/path"} {
		t.Run(issuer, func(t *testing.T) {
			values := map[string]string{"MOETOPIA_DATABASE_URL": "private-dsn", "MOETOPIA_VALKEY_ADDRESS": "localhost:6379", "MOETOPIA_ENCRYPTION_KEY": "private-key", "MOETOPIA_ISSUER": issuer}
			_, err := Read(func(k string) string { return values[k] })
			if err == nil {
				t.Fatal("accepted unsafe issuer")
			}
			if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "private-") {
				t.Fatal("configuration error leaked a secret")
			}
		})
	}
}

func TestDevelopmentAndProductionIssuer(t *testing.T) {
	values := map[string]string{"MOETOPIA_DATABASE_URL": "dsn", "MOETOPIA_VALKEY_ADDRESS": "localhost:6379", "MOETOPIA_ENCRYPTION_KEY": "key", "MOETOPIA_ISSUER": "http://account.localhost:3000"}
	get := func(k string) string { return values[k] }
	if _, err := Read(get); err != nil {
		t.Fatal(err)
	}
	values["MOETOPIA_ENV"] = "production"
	if _, err := Read(get); err == nil {
		t.Fatal("production accepted HTTP")
	}
	values["MOETOPIA_ISSUER"] = "https://account.example.com"
	if _, err := Read(get); err != nil {
		t.Fatal(err)
	}
}
