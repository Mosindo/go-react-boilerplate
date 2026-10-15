// Package mailer sends transactional email (account recovery).
package mailer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogMailer writes the message to the server log. It is the development
// fallback when SMTP is not configured and must not be used in production.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"mail_dev","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// DisabledMailer drops messages without logging their content. It is used in
// production when no SMTP server is configured so codes never reach the logs.
type DisabledMailer struct{}

func (DisabledMailer) Send(_ context.Context, to, subject, _ string) error {
	log.Printf(`{"event":"mail_disabled","subject":%q,"hint":"configure SMTP_HOST to deliver email"}`, subject)
	return nil
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type SMTPMailer struct {
	cfg SMTPConfig
}

func NewSMTP(cfg SMTPConfig) (*SMTPMailer, error) {
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.From) == "" {
		return nil, errors.New("smtp host and from address are required")
	}
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	return &SMTPMailer{cfg: cfg}, nil
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return errors.New("invalid header value")
	}
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n",
		m.cfg.From, to, subject, body)
	return smtp.SendMail(net.JoinHostPort(m.cfg.Host, m.cfg.Port), auth, m.cfg.From, []string{to}, []byte(msg))
}
