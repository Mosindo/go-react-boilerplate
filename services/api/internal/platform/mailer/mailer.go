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

// LogMailer writes the message to the server log. It is the default when no SMTP
// server is configured, for local development only.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"mail_dev_log","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

type SMTPMailer struct {
	Addr     string
	From     string
	Username string
	Password string
	Host     string
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	for _, v := range []string{to, subject, m.From} {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("mailer: header injection attempt")
		}
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(m.Addr, auth, m.From, []string{to}, []byte(msg))
}
