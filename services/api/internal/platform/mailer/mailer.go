// Package mailer sends transactional e-mails (password reset codes).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

// Mailer delivers a plain-text message.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogMailer writes messages to the server log. It is only allowed outside
// production so that local development works without an SMTP server.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// SMTPMailer sends through an SMTP relay using STARTTLS when available.
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid mail header")
	}
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	msg := "From: " + m.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body
	return smtp.SendMail(net.JoinHostPort(m.Host, m.Port), auth, m.From, []string{to}, []byte(msg))
}
