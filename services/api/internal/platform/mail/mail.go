// Package mail sends transactional e-mail (password recovery only).
package mail

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

// LogMailer prints messages to stdout. Development only: it is rejected in production by config.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev-mail] to=%s subject=%q\n%s", to, subject, body)
	return nil
}

type SMTPMailer struct {
	Host, Port, User, Password, From string
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	clean := func(s string) string { return strings.NewReplacer("\r", "", "\n", "").Replace(s) }
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		clean(m.From), clean(to), clean(subject), body)
	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Password, m.Host)
	}
	return smtp.SendMail(m.Host+":"+m.Port, auth, m.From, []string{to}, []byte(msg))
}
