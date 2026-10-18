package notifications

import (
	"context"
	"errors"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type DevelopmentSMTP struct {
	Address string
	From    string
}

func (s DevelopmentSMTP) Send(ctx context.Context, message Message) error {
	from, err := mail.ParseAddress(s.From)
	if err != nil {
		return errors.New("invalid sender configuration")
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil || to.Address != message.To || strings.ContainsAny(message.Subject, "\r\n") {
		return errors.New("invalid notification envelope")
	}
	host, _, err := net.SplitHostPort(s.Address)
	if err != nil {
		return errors.New("invalid development SMTP address")
	}
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", s.Address)
	if err != nil {
		return errors.New("development SMTP unavailable")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return errors.New("SMTP greeting failed")
	}
	defer client.Close()
	if err = client.Mail(from.Address); err != nil {
		return err
	}
	if err = client.Rcpt(to.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte("From: " + from.String() + "\r\nTo: " + to.String() + "\r\nSubject: " + message.Subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + message.Text))
	if err != nil {
		_ = writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
