// Package mailer sends transactional email (password recovery).
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

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// New returns an SMTP mailer when configured, otherwise a log mailer that is
// only suitable for local development.
func New(cfg Config, logBodies bool) Mailer {
	if cfg.Host == "" {
		return &LogMailer{logBodies: logBodies}
	}
	return &SMTPMailer{cfg: cfg}
}

type SMTPMailer struct{ cfg Config }

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
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

// LogMailer writes mails to the process log (development only).
type LogMailer struct{ logBodies bool }

func (m *LogMailer) Send(_ context.Context, to, subject, body string) error {
	if m.logBodies {
		log.Printf("[dev mailer] to=%s subject=%q\n%s", to, subject, body)
	} else {
		log.Printf("[mailer] SMTP not configured, mail to %s not sent", to)
	}
	return nil
}
