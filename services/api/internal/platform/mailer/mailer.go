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

// SMTP delivers mail through a standard SMTP relay (STARTTLS negotiated by
// net/smtp when offered by the server).
type SMTP struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSMTP(host string, port int, username, password, from string) *SMTP {
	return &SMTP{host: host, port: port, username: username, password: password, from: from}
}

func (m *SMTP) Send(_ context.Context, to, subject, body string) error {
	for _, v := range []string{to, subject} {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("invalid header value")
		}
	}
	msg := "From: " + m.from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	return smtp.SendMail(fmt.Sprintf("%s:%d", m.host, m.port), auth, m.from, []string{to}, []byte(msg))
}

// Dev prints emails to the server log. It is only wired in development so
// that password recovery can be exercised locally without an SMTP relay.
type Dev struct{}

func (Dev) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// Disabled drops emails; used in production when SMTP is not configured.
type Disabled struct{}

func (Disabled) Send(_ context.Context, _, subject, _ string) error {
	log.Printf(`{"event":"mail_disabled","subject":%q}`, subject)
	return nil
}
