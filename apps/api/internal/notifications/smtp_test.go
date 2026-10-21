package notifications

import (
	"context"
	"encoding/base64"
	"github.com/google/uuid"
	"io"
	"mime"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestSMTPVerifiedTLSAndMIME(t *testing.T) {
	for _, security := range []string{"starttls", "tls"} {
		t.Run(security, func(t *testing.T) {
			fixture := newSMTPFixture(t, security, false, true)
			sender, err := NewSMTP(SMTPConfig{Address: fixture.address, From: "Moetopia <noreply@example.test>", Security: security, Username: "private-user", Password: "private-password"})
			if err != nil {
				t.Fatal(err)
			}
			sender.tlsConfig.RootCAs = fixture.roots
			message := Message{DeliveryID: uuid.NewString(), To: "reader@example.test", Subject: "Moetopia 邮箱验证码", Text: strings.Repeat("验证码只用于本次验证。\n", 20)}
			if err := sender.Send(t.Context(), message); err != nil {
				t.Fatal(err)
			}
			select {
			case raw := <-fixture.message:
				parsed, err := mail.ReadMessage(strings.NewReader(raw))
				if err != nil {
					t.Fatal(err)
				}
				subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
				if err != nil || subject != message.Subject || parsed.Header.Get("Content-Transfer-Encoding") != "base64" {
					t.Fatal("SMTP MIME headers corrupted")
				}
				if parsed.Header.Get("Message-ID") != "<"+message.DeliveryID+"@example.test>" {
					t.Fatal("SMTP delivery identifier not preserved")
				}
				body, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, parsed.Body))
				if err != nil || string(body) != message.Text {
					t.Fatal("SMTP UTF-8 body corrupted")
				}
				if strings.Contains(raw, "private-password") || strings.Contains(raw, "private-user") {
					t.Fatal("SMTP credentials entered message")
				}
			case <-time.After(time.Second):
				t.Fatal("SMTP message missing")
			}
		})
	}
}

func TestSMTPRejectsDowngradeUntrustedCertificateAndAuthenticationFailure(t *testing.T) {
	for _, test := range []struct {
		name, serverMode  string
		trust, rejectAuth bool
		expected          string
	}{
		{"no_starttls", "none", true, false, "required STARTTLS"},
		{"untrusted", "starttls", false, false, "TLS verification failed"},
		{"auth_rejected", "starttls", true, true, "authentication failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newSMTPFixture(t, test.serverMode, test.rejectAuth, false)
			sender, err := NewSMTP(SMTPConfig{Address: fixture.address, From: "noreply@example.test", Security: "starttls", Username: "private-user", Password: "private-password"})
			if err != nil {
				t.Fatal(err)
			}
			if test.trust {
				sender.tlsConfig.RootCAs = fixture.roots
			}
			err = sender.Send(t.Context(), Message{To: "reader@example.test", Subject: "fixture", Text: "fixture"})
			if err == nil || !strings.Contains(err.Error(), test.expected) {
				t.Fatal("SMTP did not fail closed")
			}
			if strings.Contains(err.Error(), "private-user") || strings.Contains(err.Error(), "private-password") {
				t.Fatal("SMTP error disclosed credentials")
			}
			select {
			case <-fixture.message:
				t.Fatal("SMTP failure still submitted mail")
			default:
			}
		})
	}
}

func TestSMTPRejectsUnsafeConfigurationAndEnvelope(t *testing.T) {
	base := SMTPConfig{Address: "smtp.example.test:587", From: "noreply@example.test", Security: "starttls"}
	for _, change := range []func(*SMTPConfig){
		func(c *SMTPConfig) { c.Security = "none" }, func(c *SMTPConfig) { c.Security = "typo" }, func(c *SMTPConfig) { c.Address = "smtp.example.test" }, func(c *SMTPConfig) { c.Address = "smtp.example.test:0" }, func(c *SMTPConfig) { c.From = "noreply@example.test\r\nBcc: victim@example.test" }, func(c *SMTPConfig) { c.Username = "private-user" }, func(c *SMTPConfig) { c.Password = "private-password" },
	} {
		c := base
		change(&c)
		if _, err := NewSMTP(c); err == nil {
			t.Fatal("unsafe SMTP configuration accepted")
		}
	}
	sender, err := NewSMTP(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range []Message{{To: "reader@example.test", Subject: "test\r\nBcc: victim@example.test"}, {To: "reader@example.test\r\nRCPT TO:<victim@example.test>", Subject: "test"}} {
		if err := sender.Send(t.Context(), message); err == nil || !strings.Contains(err.Error(), "envelope") {
			t.Fatal("SMTP header injection accepted")
		}
	}
}

func TestSMTPCancellationClosesBlockedConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		_, _ = io.Copy(io.Discard, conn)
	}()
	sender, err := NewSMTP(SMTPConfig{Address: listener.Addr().String(), From: "noreply@example.test", Security: "none", AllowPlain: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := sender.Send(ctx, Message{To: "reader@example.test", Subject: "fixture", Text: "fixture"}); err == nil || time.Since(started) > time.Second {
		t.Fatal("SMTP ignored context timeout")
	}
	<-closed
}
