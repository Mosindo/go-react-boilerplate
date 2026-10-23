// Package mailer sends transactional email (password reset codes).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogMailer prints messages to the log. Development only: it exposes reset codes.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"mail_dev","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

type SMTPMailer struct {
	Host, Port, Username, Password, From string
}

func (m SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject+m.From, "\r\n") {
		return fmt.Errorf("mailer: header injection attempt")
	}
	msg := strings.Join([]string{
		"From: " + m.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(m.Host+":"+m.Port, auth, m.From, []string{to}, []byte(msg)) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
		return fmt.Errorf("mailer: smtp timeout")
	}
}
