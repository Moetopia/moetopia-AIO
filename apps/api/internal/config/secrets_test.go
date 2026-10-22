package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretFilesBoundedAndUnambiguous(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte(" secret with spaces \r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := map[string]string{"MOETOPIA_ENCRYPTION_KEY_FILE": path}
	get := func(key string) string { return settings[key] }
	values, err := loadSecrets(get)
	if err != nil || values["MOETOPIA_ENCRYPTION_KEY"] != " secret with spaces " {
		t.Fatal("secret file content was changed beyond line endings")
	}
	settings["MOETOPIA_ENCRYPTION_KEY"] = "inline"
	if _, err := loadSecrets(get); err == nil || strings.Contains(err.Error(), "inline") {
		t.Fatal("ambiguous secret input accepted or exposed")
	}
	delete(settings, "MOETOPIA_ENCRYPTION_KEY")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 16385)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSecrets(get); err == nil {
		t.Fatal("unbounded secret accepted")
	}
	settings["MOETOPIA_ENCRYPTION_KEY_FILE"] = filepath.Join(t.TempDir(), "missing")
	if _, err := loadSecrets(get); err == nil || strings.Contains(err.Error(), settings["MOETOPIA_ENCRYPTION_KEY_FILE"]) {
		t.Fatal("missing secret file accepted or path exposed")
	}
}
