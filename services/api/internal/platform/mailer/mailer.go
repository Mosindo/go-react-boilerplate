// Package mailer sends transactional emails (account recovery).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// Mailer delivers a plain text email.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogMailer writes emails to the log. Development only: it never reaches a real inbox.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"mail_dev","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// SMTPMailer delivers email through an SMTP relay (STARTTLS when supported by the server).
type SMTPMailer struct {
	Addr     string
	Username string
	Password string
	From     string
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	host := m.Addr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, host)
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	return smtp.SendMail(m.Addr, auth, m.From, []string{to}, []byte(msg))
}
