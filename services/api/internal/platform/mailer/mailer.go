// Package mailer sends transactional e-mail (password reset codes).
package mailer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPSender delivers through an SMTP relay (STARTTLS is negotiated by net/smtp when offered).
type SMTPSender struct {
	Host, Username, Password, From string
	Port                           int
}

func (s SMTPSender) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return errors.New("invalid header value")
	}
	msg := "From: " + s.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	return smtp.SendMail(addr, auth, s.From, []string{to}, []byte(msg))
}

// LogSender prints messages to the server log. Development only: config refuses to
// start in production without SMTP.
type LogSender struct{}

func (LogSender) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev mailer] to=%s subject=%q\n%s", to, subject, body)
	return nil
}
