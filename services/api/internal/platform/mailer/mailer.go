// Package mailer sends transactional e-mails (password reset codes).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Mailer is implemented by SMTP and by the development log mailer.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPConfig configures a plain SMTP relay (STARTTLS when offered).
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

func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid mail header")
	}
	msg := strings.Join([]string{
		"From: " + m.cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	return smtp.SendMail(net.JoinHostPort(m.cfg.Host, m.cfg.Port), auth, m.cfg.From, []string{to}, []byte(msg))
}

// LogMailer prints e-mails to the server log. It is only wired outside
// production so that password reset can be tested locally without SMTP.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// DisabledMailer drops e-mails; used in production when SMTP is missing so
// secrets never end up in logs.
type DisabledMailer struct{}

func (DisabledMailer) Send(_ context.Context, to, subject, _ string) error {
	log.Printf(`{"event":"mail_dropped","reason":"smtp not configured","subject":%q}`, subject)
	return nil
}
