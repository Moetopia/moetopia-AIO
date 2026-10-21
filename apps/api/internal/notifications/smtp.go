package notifications

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type SMTPConfig struct {
	Address, From, Security, Username, Password string
	AllowPlain                                  bool
}

type SMTP struct {
	config    SMTPConfig
	host      string
	from      *mail.Address
	tlsConfig *tls.Config
}

func NewSMTP(config SMTPConfig) (*SMTP, error) {
	host, port, err := net.SplitHostPort(config.Address)
	if err != nil || host == "" || strings.ContainsAny(host, "\r\n\x00") {
		return nil, errors.New("MOETOPIA_SMTP_ADDRESS requires host:port")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return nil, errors.New("MOETOPIA_SMTP_ADDRESS requires a valid port")
	}
	from, err := mail.ParseAddress(config.From)
	if err != nil || strings.ContainsAny(config.From, "\r\n\x00") {
		return nil, errors.New("MOETOPIA_SMTP_FROM requires a valid mailbox")
	}
	if config.Security != "starttls" && config.Security != "tls" && !(config.AllowPlain && config.Security == "none") {
		return nil, errors.New("MOETOPIA_SMTP_SECURITY must be starttls or tls; none is limited to development/test")
	}
	if (config.Username == "") != (config.Password == "") || strings.ContainsAny(config.Username+config.Password, "\x00\r\n") {
		return nil, errors.New("MOETOPIA_SMTP_USERNAME and MOETOPIA_SMTP_PASSWORD must be configured together")
	}
	if config.Security == "none" && config.Username != "" {
		return nil, errors.New("SMTP authentication requires TLS")
	}
	return &SMTP{config: config, host: host, from: from, tlsConfig: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}}, nil
}

func (s *SMTP) Send(ctx context.Context, message Message) error {
	to, err := mail.ParseAddress(message.To)
	if err != nil || to.Address != message.To || strings.ContainsAny(message.To+message.Subject, "\r\n\x00") {
		return errors.New("invalid notification envelope")
	}
	if message.DeliveryID == "" {
		message.DeliveryID = uuid.NewString()
	}
	if _, err := uuid.Parse(message.DeliveryID); err != nil {
		return errors.New("invalid notification delivery identifier")
	}
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", s.config.Address)
	if err != nil {
		return errors.New("SMTP connection unavailable")
	}
	defer conn.Close()
	rawConn := conn
	stop := context.AfterFunc(ctx, func() { _ = rawConn.Close() })
	defer stop()
	deadline := time.Now().Add(15 * time.Second)
	if limit, ok := ctx.Deadline(); ok && limit.Before(deadline) {
		deadline = limit
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return errors.New("SMTP timeout could not be configured")
	}
	if s.config.Security == "tls" {
		secured := tls.Client(conn, s.tlsConfig)
		if err := secured.HandshakeContext(ctx); err != nil {
			return errors.New("SMTP TLS verification failed")
		}
		conn = secured
	}
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return errors.New("SMTP greeting failed")
	}
	defer client.Close()
	if s.config.Security == "starttls" {
		if available, _ := client.Extension("STARTTLS"); !available {
			return errors.New("SMTP server does not support required STARTTLS")
		}
		if err := client.StartTLS(s.tlsConfig); err != nil {
			return errors.New("SMTP STARTTLS verification failed")
		}
	}
	if s.config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.config.Username, s.config.Password, s.host)); err != nil {
			return errors.New("SMTP authentication failed")
		}
	}
	if err = client.Mail(s.from.Address); err != nil {
		return errors.New("SMTP sender rejected")
	}
	if err = client.Rcpt(to.Address); err != nil {
		return errors.New("SMTP recipient rejected")
	}
	writer, err := client.Data()
	if err != nil {
		return errors.New("SMTP message rejected")
	}
	_, err = writer.Write([]byte(s.encode(message, to)))
	if err != nil {
		_ = writer.Close()
		return errors.New("SMTP message transfer failed")
	}
	if err = writer.Close(); err != nil {
		return errors.New("SMTP message acceptance failed")
	}
	// Once DATA is accepted, a failed QUIT must not cause duplicate notification retries.
	_ = client.Quit()
	return nil
}

func (s *SMTP) encode(message Message, to *mail.Address) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(message.Text))
	var body strings.Builder
	for len(encoded) > 76 {
		body.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	body.WriteString(encoded + "\r\n")
	domain := s.from.Address[strings.LastIndex(s.from.Address, "@")+1:]
	return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@%s>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", s.from.String(), to.String(), mime.QEncoding.Encode("UTF-8", message.Subject), time.Now().UTC().Format(time.RFC1123Z), message.DeliveryID, domain, body.String())
}
