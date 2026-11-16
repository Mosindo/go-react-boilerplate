// Package mailer sends transactional emails (password reset codes).
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

// LogMailer prints emails to the server log. Only used outside production so
// developers can complete the password reset flow without an SMTP server.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_email","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// DisabledMailer drops emails. Used in production when SMTP is not configured,
// so secrets such as reset codes never end up in logs.
type DisabledMailer struct{}

func (DisabledMailer) Send(_ context.Context, _, subject, _ string) error {
	log.Printf(`{"event":"email_dropped","reason":"smtp_not_configured","subject":%q}`, subject)
	return nil
}

type SMTPMailer struct {
	Addr     string
	Host     string
	Username string
	Password string
	From     string
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n"
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(m.Addr, auth, m.From, []string{to}, []byte(msg))
}
