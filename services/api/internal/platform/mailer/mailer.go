// Package mailer sends transactional email (password recovery codes).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"sync"

	"example.com/api/internal/platform/config"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// New returns an SMTP mailer when configured. Without SMTP settings, mail is
// only logged in development; in production it is dropped with a warning so
// that secrets never reach logs.
func New(cfg config.Config) Mailer {
	if cfg.SMTP.Enabled() {
		return &smtpMailer{cfg: cfg.SMTP}
	}
	return &logMailer{logBody: !cfg.IsProduction()}
}

type smtpMailer struct{ cfg config.SMTP }

func (m *smtpMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.cfg.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.cfg.User != "" {
		auth = smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	}
	return smtp.SendMail(m.cfg.Host+":"+m.cfg.Port, auth, m.cfg.From, []string{to}, []byte(msg))
}

type logMailer struct{ logBody bool }

func (m *logMailer) Send(_ context.Context, to, subject, body string) error {
	if m.logBody {
		log.Printf("mailer(dev): to=%s subject=%q body=%q", to, subject, body)
	} else {
		log.Printf("mailer: SMTP is not configured; dropping %q mail", subject)
	}
	return nil
}

// Memory records messages for tests.
type Memory struct {
	mu   sync.Mutex
	sent []Message
}

type Message struct{ To, Subject, Body string }

func (m *Memory) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, Message{to, subject, body})
	return nil
}

func (m *Memory) Messages() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Message(nil), m.sent...)
}
