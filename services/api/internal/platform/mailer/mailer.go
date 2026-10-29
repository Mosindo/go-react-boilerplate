// Package mailer sends transactional emails (password recovery).
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

type SMTP struct {
	Host, Port, Username, Password, From string
}

func (m SMTP) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(m.Host+":"+m.Port, auth, m.From, []string{to}, []byte(msg))
}

// Log writes emails to the process log. It is only meant for local development
// where no SMTP server is configured; production requires SMTP (see config).
type Log struct{}

func (Log) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev mailer] to=%s subject=%q\n%s", to, subject, body)
	return nil
}
