package restoration

import "testing"

func TestRestoreTargetGuard(t *testing.T) {
	for _, value := range []string{"postgres://localhost/moetopia", "postgres://localhost/moetopia_dev", "postgres://localhost/moetopia_restore_bad", "postgres://localhost/moetopia_restore_0123456789abcdef/extra", "host=localhost dbname=moetopia_restore_0123456789abcdef", "file:///moetopia_restore_0123456789abcdef", "postgres://localhost/moetopia_restore_0123456789abcdef?dbname=moetopia_dev"} {
		if SafeTarget(value) {
			t.Fatal("unsafe restore target accepted")
		}
	}
	if !SafeTarget("postgres://localhost/moetopia_restore_0123456789abcdef") {
		t.Fatal("valid scratch restore target rejected")
	}
}
