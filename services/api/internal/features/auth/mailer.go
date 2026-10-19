package auth

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// Mailer delivers account-recovery codes.
type Mailer interface {
	SendPasswordReset(ctx context.Context, to, code string) error
}

// LogMailer is the development fallback: it writes the code to the server log.
// It must not be used in production because it leaks the recovery code.
type LogMailer struct{}

func (LogMailer) SendPasswordReset(_ context.Context, to, code string) error {
	log.Printf(`{"event":"password_reset_code","to":%q,"code":%q,"note":"development mailer"}`, to, code)
	return nil
}

// DisabledMailer is used in production when SMTP is not configured. Recovery
// requests are accepted (so they do not reveal anything) but nothing is sent.
type DisabledMailer struct{}

func (DisabledMailer) SendPasswordReset(context.Context, string, string) error {
	log.Printf(`{"event":"password_reset_not_sent","reason":"SMTP is not configured"}`)
	return nil
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (c SMTPConfig) Enabled() bool {
	return c.Host != "" && c.From != ""
}

type SMTPMailer struct {
	cfg SMTPConfig
}

func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) SendPasswordReset(_ context.Context, to, code string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(m.cfg.From, "\r\n") {
		return fmt.Errorf("invalid mail header value")
	}
	body := "Subject: Votre code de récupération\r\n" +
		"From: " + m.cfg.From + "\r\n" +
		"To: " + to + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Votre code de récupération : " + code + "\r\n\r\n" +
		"Il expire dans 1 heure. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.\r\n"
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	return smtp.SendMail(m.cfg.Host+":"+m.cfg.Port, auth, m.cfg.From, []string{to}, []byte(body))
}
