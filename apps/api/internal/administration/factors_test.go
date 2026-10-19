package administration

import (
	"testing"
	"time"
)

func TestTOTPRFC6238VectorAndReplayBoundary(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	counter, ok := totpCounter(secret, "287082", -1, time.Unix(59, 0))
	if !ok || counter != 1 {
		t.Fatal("RFC 6238 SHA-1 vector failed")
	}
	if _, ok := totpCounter(secret, "287082", counter, time.Unix(59, 0)); ok {
		t.Fatal("consumed counter was replayable")
	}
	if _, ok := totpCounter(secret, "287082", -1, time.Unix(89, 0)); !ok {
		t.Fatal("one-period backward skew rejected")
	}
	if _, ok := totpCounter(secret, "287082", -1, time.Unix(150, 0)); ok {
		t.Fatal("expired counter accepted outside clock skew")
	}
}
