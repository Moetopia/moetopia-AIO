package cryptobox

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestEncryptionBindsCiphertextToPurpose(t *testing.T) {
	box, err := New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := box.Seal([]byte("sensitive state"), "grant:a")
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := box.Open(payload, "grant:a")
	if err != nil || string(plaintext) != "sensitive state" {
		t.Fatal("round trip failed")
	}
	if _, err := box.Open(payload, "grant:b"); err == nil {
		t.Fatal("accepted another record's ciphertext")
	}
	payload[len(payload)-1] ^= 1
	if _, err := box.Open(payload, "grant:a"); err == nil {
		t.Fatal("accepted tampered ciphertext")
	}
}

func TestRejectInvalidKeysAndTruncatedPayload(t *testing.T) {
	for _, key := range []string{"", "invalid", base64.StdEncoding.EncodeToString([]byte("short"))} {
		if _, err := New(key); err == nil {
			t.Fatal("accepted invalid key")
		}
	}
	box, err := New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := box.Open([]byte{1}, "grant"); err == nil {
		t.Fatal("accepted truncated payload")
	}
}
