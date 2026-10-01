// Package mailer sends transactional email (account recovery).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTP delivers through a standard SMTP relay with PLAIN auth (STARTTLS negotiated by net/smtp).
type SMTP struct{ host, port, user, pass, from string }

func NewSMTP(host, port, user, pass, from string) *SMTP {
	return &SMTP{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *SMTP) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("mailer: header injection attempt")
	}
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}
	msg := "From: " + s.from + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	return smtp.SendMail(s.host+":"+s.port, auth, s.from, []string{to}, []byte(msg))
}

// Log writes messages to the server log. Development only: it exposes recovery codes.
type Log struct{}

func (Log) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[mailer:dev] to=%s subject=%q body=%q", to, subject, body)
	return nil
}
