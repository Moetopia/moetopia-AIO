package config

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Environment    string
	Listen         string
	Issuer         string
	DatabaseURL    string
	ValkeyAddress  string
	ValkeyPassword string
	EncryptionKey  string
	AccountOrigin  string
	AdminOrigin    string
	SMTPAddress    string
	SMTPFrom       string
}

func Load() (Config, error) {
	return Read(os.Getenv)
}
func (c Config) RateNamespace() string {
	hash := sha256.Sum256([]byte(c.Environment + "\n" + c.Issuer + "\n" + c.DatabaseURL))
	return "moetopia:" + hex.EncodeToString(hash[:16]) + ":"
}

func Read(get func(string) string) (Config, error) {
	c := Config{
		Environment:    value(get, "MOETOPIA_ENV", "development"),
		Listen:         value(get, "MOETOPIA_LISTEN", "127.0.0.1:8080"),
		Issuer:         strings.TrimRight(get("MOETOPIA_ISSUER"), "/"),
		DatabaseURL:    get("MOETOPIA_DATABASE_URL"),
		ValkeyAddress:  get("MOETOPIA_VALKEY_ADDRESS"),
		ValkeyPassword: get("MOETOPIA_VALKEY_PASSWORD"),
		EncryptionKey:  get("MOETOPIA_ENCRYPTION_KEY"),
		AccountOrigin:  value(get, "MOETOPIA_ACCOUNT_ORIGIN", strings.TrimRight(get("MOETOPIA_ISSUER"), "/")),
		AdminOrigin:    strings.TrimRight(get("MOETOPIA_ADMIN_ORIGIN"), "/"),
		SMTPAddress:    get("MOETOPIA_SMTP_ADDRESS"),
		SMTPFrom:       get("MOETOPIA_SMTP_FROM"),
	}
	if c.Environment != "development" && c.Environment != "production" && c.Environment != "test" {
		return c, errors.New("MOETOPIA_ENV must be development, test or production")
	}
	for _, field := range []struct{ name, value string }{
		{"MOETOPIA_DATABASE_URL", c.DatabaseURL}, {"MOETOPIA_VALKEY_ADDRESS", c.ValkeyAddress},
		{"MOETOPIA_ENCRYPTION_KEY", c.EncryptionKey}, {"MOETOPIA_ISSUER", c.Issuer},
	} {
		if field.value == "" {
			return c, errors.New(field.name + " is required")
		}
	}
	u, err := url.Parse(c.Issuer)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return c, errors.New("MOETOPIA_ISSUER must be an origin without credentials, path, query or fragment")
	}
	if u.Scheme != "https" && !(c.Environment != "production" && u.Scheme == "http" && localHost(u.Hostname())) {
		return c, errors.New("MOETOPIA_ISSUER requires HTTPS; development HTTP is limited to loopback hosts")
	}
	if c.AccountOrigin != c.Issuer {
		return c, errors.New("MOETOPIA_ACCOUNT_ORIGIN must match the current account issuer origin")
	}
	if c.AdminOrigin != "" {
		admin, err := url.Parse(c.AdminOrigin)
		if err != nil || admin.Host == "" || admin.User != nil || admin.RawQuery != "" || admin.Fragment != "" || admin.Path != "" || c.AdminOrigin == c.Issuer {
			return c, errors.New("MOETOPIA_ADMIN_ORIGIN must be an independent origin without credentials, path, query or fragment")
		}
		if admin.Scheme != "https" && !(c.Environment != "production" && admin.Scheme == "http" && localHost(admin.Hostname())) {
			return c, errors.New("MOETOPIA_ADMIN_ORIGIN requires HTTPS; development HTTP is limited to loopback hosts")
		}
	}
	return c, nil
}

func localHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost")
}

func value(get func(string) string, key, fallback string) string {
	if v := get(key); v != "" {
		return v
	}
	return fallback
}
