package notifications

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"io"
	"math/big"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

type smtpFixture struct {
	address string
	roots   *x509.CertPool
	message chan string
}

func smtpCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, IsCA: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, roots
}

func newSMTPFixture(t *testing.T, security string, rejectAuth, dropQuit bool) smtpFixture {
	t.Helper()
	certificate, roots := smtpCertificate(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	message := make(chan string, 1)
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		secured := security == "tls"
		if secured {
			conn = tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
			if err := conn.(*tls.Conn).Handshake(); err != nil {
				return
			}
		}
		_, _ = io.WriteString(conn, "220 localhost ESMTP\r\n")
		reader := textproto.NewReader(bufio.NewReader(conn))
		for {
			line, err := reader.ReadLine()
			if err != nil {
				return
			}
			verb := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
			switch verb {
			case "EHLO":
				_, _ = io.WriteString(conn, "250-localhost\r\n")
				if security == "starttls" && !secured {
					_, _ = io.WriteString(conn, "250-STARTTLS\r\n")
				}
				_, _ = io.WriteString(conn, "250 AUTH PLAIN\r\n")
			case "HELO":
				_, _ = io.WriteString(conn, "250 localhost\r\n")
			case "STARTTLS":
				_, _ = io.WriteString(conn, "220 upgrade\r\n")
				conn = tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
				if err := conn.(*tls.Conn).Handshake(); err != nil {
					return
				}
				secured = true
				reader = textproto.NewReader(bufio.NewReader(conn))
			case "AUTH":
				fields := strings.Fields(line)
				valid := secured && len(fields) == 3 && fields[1] == "PLAIN"
				if valid {
					raw, err := base64.StdEncoding.DecodeString(fields[2])
					valid = err == nil && string(raw) == "\x00private-user\x00private-password"
				}
				if valid && !rejectAuth {
					_, _ = io.WriteString(conn, "235 authenticated\r\n")
				} else {
					_, _ = io.WriteString(conn, "535 rejected\r\n")
				}
			case "MAIL", "RCPT":
				_, _ = io.WriteString(conn, "250 accepted\r\n")
			case "DATA":
				_, _ = io.WriteString(conn, "354 data\r\n")
				raw, err := reader.ReadDotBytes()
				if err != nil {
					return
				}
				message <- string(raw)
				_, _ = io.WriteString(conn, "250 queued\r\n")
				if dropQuit {
					return
				}
			case "QUIT":
				_, _ = io.WriteString(conn, "221 goodbye\r\n")
				return
			default:
				_, _ = io.WriteString(conn, "500 unsupported\r\n")
			}
		}
	}()
	t.Cleanup(func() { _ = listener.Close(); <-done })
	return smtpFixture{listener.Addr().String(), roots, message}
}
