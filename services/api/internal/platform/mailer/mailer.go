// Package mailer sends transactional email (password recovery).
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

// SMTPMailer delivers through an SMTP relay (STARTTLS when offered, via net/smtp).
type SMTPMailer struct {
	Host, Username, Password, From string
	Port                           int
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(fmt.Sprintf("%s:%d", m.Host, m.Port), auth, m.From, []string{to}, []byte(msg))
}

// LogMailer writes the message to the server log. Development only: it is
// refused by config validation in production.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev mailer] to=%s subject=%q\n%s", to, subject, body)
	return nil
}
