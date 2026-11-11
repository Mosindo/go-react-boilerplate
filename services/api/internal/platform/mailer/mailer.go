// Package mailer sends transactional e-mails (password reset codes).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"example.com/api/internal/platform/config"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// New returns an SMTP mailer when SMTP is configured. Otherwise, outside of
// production, it returns a development mailer that writes e-mails to the
// server log so the reset flow can be exercised locally. In production without
// SMTP it returns a mailer that fails loudly.
func New(cfg config.Config) Mailer {
	if cfg.SMTP.Enabled() {
		return &SMTPMailer{cfg: cfg.SMTP}
	}
	if cfg.IsProduction() {
		return disabledMailer{}
	}
	return LogMailer{}
}

type SMTPMailer struct {
	cfg config.SMTPConfig
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	msg := "From: " + m.cfg.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body
	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg))
}

// LogMailer is for local development only.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

type disabledMailer struct{}

func (disabledMailer) Send(context.Context, string, string, string) error {
	return fmt.Errorf("e-mail delivery is not configured (set SMTP_HOST and SMTP_FROM)")
}
