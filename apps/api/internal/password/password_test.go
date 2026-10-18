package password

import (
	"context"
	"strings"
	"testing"
)

func TestPasswordHashAndBounds(t *testing.T) {
	ctx := context.Background()
	value := "a long passphrase with spaces"
	hash, err := Hash(ctx, value)
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(ctx, value, hash) || Verify(ctx, value+"x", hash) {
		t.Fatal("password verification mismatch")
	}
	if _, err := Hash(ctx, "short"); err == nil {
		t.Fatal("short password accepted")
	}
	if Verify(ctx, strings.Repeat("a", 257), hash) || Verify(ctx, value, "$argon2id$v=19$m=999999999,t=99,p=99$invalid$hash") {
		t.Fatal("unbounded parameters accepted")
	}
	other, err := Hash(ctx, value)
	if err != nil || other == hash {
		t.Fatal("password salts must be unique")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	capacity <- struct{}{}
	capacity <- struct{}{}
	if _, err := derive(cancelled, value, make([]byte, 16)); err == nil {
		t.Fatal("cancelled waiter continued")
	}
	<-capacity
	<-capacity
}
