// Package mail sends transactional emails (password reset).
package mail

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"example.com/api/internal/platform/config"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// New returns an SMTP mailer when SMTP is configured, otherwise a development
// mailer that prints messages to the server log. Production configuration
// requires SMTP (see config.Load), so the log mailer never runs there.
func New(cfg config.SMTP) Mailer {
	if cfg.Configured() {
		return &SMTPMailer{cfg: cfg}
	}
	return LogMailer{}
}

type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev-mail] to=%s subject=%q\n%s", to, subject, body)
	return nil
}

type SMTPMailer struct {
	cfg config.SMTP
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("mail: header injection attempt")
	}
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	msg := strings.Join([]string{
		"From: " + m.cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if m.cfg.User != "" {
		auth = smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	}

	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg)) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
		return fmt.Errorf("mail: smtp timeout")
	}
}
