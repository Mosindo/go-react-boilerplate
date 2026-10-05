// Package mailer sends transactional email (password recovery).
package mailer

import (
	"context"
	"errors"
	"log"
	"net"
	"net/smtp"
	"strings"
	"sync"
)

// Mailer delivers a plain-text email.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type smtpMailer struct{ cfg SMTPConfig }

func NewSMTP(cfg SMTPConfig) Mailer { return &smtpMailer{cfg: cfg} }

func (m *smtpMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject+m.cfg.From, "\r\n") {
		return errors.New("mailer: header injection attempt")
	}
	msg := "From: " + m.cfg.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	// net/smtp upgrades to STARTTLS when the server offers it; PlainAuth refuses to send credentials in clear text.
	return smtp.SendMail(net.JoinHostPort(m.cfg.Host, m.cfg.Port), auth, m.cfg.From, []string{to}, []byte(msg))
}

type logMailer struct{}

// NewLog returns a development mailer that prints messages to the server log.
// It must never be used in production because messages contain recovery links.
func NewLog() Mailer { return logMailer{} }

func (logMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev mailer] to=%s subject=%q\n%s", to, subject, body)
	return nil
}

// Memory records messages in tests.
type Memory struct {
	mu   sync.Mutex
	sent []Message
}

type Message struct{ To, Subject, Body string }

func (m *Memory) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, Message{To: to, Subject: subject, Body: body})
	return nil
}

// Messages returns a copy of everything sent so far.
func (m *Memory) Messages() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Message(nil), m.sent...)
}
