// Package mailer sends transactional emails (password recovery).
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

// LogMailer prints emails to the server log. Development only: it exposes recovery codes in logs.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev-mailer] to=%s subject=%q\n%s", to, subject, body)
	return nil
}

type SMTPMailer struct {
	Host, User, Pass, From string
	Port                   int
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Pass, m.Host)
	}
	return smtp.SendMail(fmt.Sprintf("%s:%d", m.Host, m.Port), auth, m.From, []string{to}, []byte(msg))
}
