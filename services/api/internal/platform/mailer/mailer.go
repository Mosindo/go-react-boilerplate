// Package mailer sends transactional e-mails (account recovery codes).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPConfig is read from SMTP_* environment variables.
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

func NewSMTP(cfg SMTPConfig) *SMTPMailer { return &SMTPMailer{cfg: cfg} }

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.cfg.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n"
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	// net/smtp upgrades to STARTTLS automatically when the server offers it.
	return smtp.SendMail(net.JoinHostPort(m.cfg.Host, m.cfg.Port), auth, m.cfg.From, []string{to}, []byte(msg))
}

// DevLogMailer prints e-mails to stdout. It is only wired when APP_ENV is not
// "production", so recovery codes never end up in production logs.
type DevLogMailer struct{}

func (DevLogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// DisabledMailer is used in production when SMTP is not configured.
type DisabledMailer struct{}

func (DisabledMailer) Send(_ context.Context, _, subject, _ string) error {
	log.Printf(`{"event":"mail_not_sent","reason":"smtp not configured","subject":%q}`, subject)
	return nil
}
