// Package mail sends transactional e-mails (password reset codes).
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

// LogMailer writes messages to the log. It is only suitable for local
// development because the body contains secrets (one-time codes).
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev-mail] to=%s subject=%q body=%q", to, subject, body)
	return nil
}

// DiscardMailer drops messages; used when no SMTP server is configured in production.
type DiscardMailer struct{}

func (DiscardMailer) Send(context.Context, string, string, string) error { return nil }

type SMTPMailer struct {
	Addr     string
	Host     string
	Username string
	Password string
	From     string
}

func NewSMTP(host, port, username, password, from string) *SMTPMailer {
	return &SMTPMailer{Addr: host + ":" + port, Host: host, Username: username, Password: password, From: from}
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("mail: header injection attempt")
	}
	msg := "From: " + m.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" + body + "\r\n"
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(m.Addr, auth, m.From, []string{to}, []byte(msg))
}
