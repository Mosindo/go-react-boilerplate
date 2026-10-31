// Package mail delivers transactional email (password reset codes).
package mail

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

// New returns an SMTP mailer when configured, otherwise a development mailer
// that writes messages to the server log.
func New(cfg config.SMTPConfig) Mailer {
	if cfg.Host == "" {
		return logMailer{}
	}
	return smtpMailer{cfg: cfg}
}

type logMailer struct{}

func (logMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev-mail] to=%s subject=%q body=%q", to, subject, body)
	return nil
}

type smtpMailer struct{ cfg config.SMTPConfig }

func (m smtpMailer) Send(_ context.Context, to, subject, body string) error {
	for _, v := range []string{to, subject, m.cfg.From} {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("invalid header value")
		}
	}
	msg := "From: " + m.cfg.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	return smtp.SendMail(m.cfg.Host+":"+m.cfg.Port, auth, m.cfg.From, []string{to}, []byte(msg))
}
