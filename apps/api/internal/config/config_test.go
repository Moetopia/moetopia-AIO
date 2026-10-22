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

func TestAdministratorOriginIsolation(t *testing.T) {
	values := map[string]string{"MOETOPIA_DATABASE_URL": "dsn", "MOETOPIA_VALKEY_ADDRESS": "localhost:6379", "MOETOPIA_ENCRYPTION_KEY": "key", "MOETOPIA_ISSUER": "https://account.example.test"}
	for _, origin := range []string{"https://account.example.test", "https://admin.example.test/path", "https://user:secret@admin.example.test", "http://192.168.101.41:3001", "https://admin.example.test?token=secret"} {
		values["MOETOPIA_ADMIN_ORIGIN"] = origin
		if _, err := Read(func(k string) string { return values[k] }); err == nil {
			t.Fatal("unsafe administrator origin accepted")
		}
	}
	values["MOETOPIA_ADMIN_ORIGIN"] = "https://admin.example.test"
	if _, err := Read(func(k string) string { return values[k] }); err != nil {
		t.Fatal(err)
	}
}

func TestExternalProviderCredentialPairs(t *testing.T) {
	for _, provider := range []string{"GITHUB", "GOOGLE"} {
		values := map[string]string{"MOETOPIA_DATABASE_URL": "dsn", "MOETOPIA_VALKEY_ADDRESS": "localhost:6379", "MOETOPIA_ENCRYPTION_KEY": "key", "MOETOPIA_ISSUER": "https://account.example.test"}
		get := func(k string) string { return values[k] }
		values["MOETOPIA_"+provider+"_CLIENT_SECRET"] = "private-secret"
		if _, err := Read(get); err == nil || strings.Contains(err.Error(), "private-secret") {
			t.Fatal("incomplete provider configuration accepted or exposed")
		}
		values["MOETOPIA_"+provider+"_CLIENT_ID"] = "client"
		if _, err := Read(get); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProxyCIDRsMustBeExplicit(t *testing.T) {
	values := map[string]string{"MOETOPIA_DATABASE_URL": "dsn", "MOETOPIA_VALKEY_ADDRESS": "localhost:6379", "MOETOPIA_ENCRYPTION_KEY": "key", "MOETOPIA_ISSUER": "https://account.example.test"}
	get := func(key string) string { return values[key] }
	for _, value := range []string{"*", "0.0.0.0/0", "::/0", "192.0.2.1", "192.0.2.0/33"} {
		values["MOETOPIA_TRUSTED_PROXY_CIDRS"] = value
		if _, err := Read(get); err == nil {
			t.Fatal("unsafe proxy trust accepted")
		}
	}
	values["MOETOPIA_TRUSTED_PROXY_CIDRS"] = "172.31.240.0/28, 2001:db8::/64"
	cfg, err := Read(get)
	if err != nil || len(cfg.TrustedProxies) != 2 {
		t.Fatal("explicit proxy networks rejected")
	}
}
