// Package mailer sends transactional email (account recovery only).
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

// SMTPMailer delivers through an SMTP relay using PLAIN auth when credentials are set.
type SMTPMailer struct {
	Host, Username, Password, From string
	Port                           int
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(addr, auth, m.From, []string{to}, []byte(msg))
}

// LogMailer is the development fallback: it prints the message instead of sending it.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev-mail] to=%s subject=%q body=%q", to, subject, body)
	return nil
}

// Discard drops every message (used when neither SMTP nor dev logging is enabled).
type Discard struct{}

func (Discard) Send(context.Context, string, string, string) error { return nil }
