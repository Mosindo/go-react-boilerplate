// Package mailer sends transactional e-mails. Without SMTP configuration it logs the message
// (development only) instead of sending it.
package mailer

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

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// New returns an SMTP mailer when Host is set, a log mailer otherwise.
func New(cfg SMTPConfig) Mailer {
	if cfg.Host == "" {
		return logMailer{}
	}
	return smtpMailer{cfg: cfg}
}

type logMailer struct{}

func (logMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"mail_not_sent_no_smtp","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

type smtpMailer struct{ cfg SMTPConfig }

func (m smtpMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	msg := "From: " + m.cfg.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	return smtp.SendMail(m.cfg.Host+":"+m.cfg.Port, auth, m.cfg.From, []string{to}, []byte(msg))
}
