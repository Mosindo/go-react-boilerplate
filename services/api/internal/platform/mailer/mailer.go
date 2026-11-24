// Package mailer sends transactional emails (account recovery). Without SMTP
// configuration, development builds log messages to stdout instead.
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

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogMailer prints emails to the log. Only meant for local development.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_email","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// DiscardMailer drops emails. Used in production when SMTP is not configured
// so that secrets such as reset codes never end up in logs.
type DiscardMailer struct{}

func (DiscardMailer) Send(context.Context, string, string, string) error {
	log.Printf(`{"event":"email_dropped","reason":"smtp not configured"}`)
	return nil
}

// SMTPMailer delivers email through an SMTP relay (STARTTLS when offered).
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid email header")
	}
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	msg := strings.Join([]string{
		"From: " + m.From,
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	return smtp.SendMail(net.JoinHostPort(m.Host, m.Port), auth, m.From, []string{to}, []byte(msg))
}
