// Package mail sends transactional emails (currently only password recovery).
package mail

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

// LogMailer writes emails to the server log. It is meant for local development only.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// SMTPMailer delivers email through an SMTP relay using STARTTLS when offered.
type SMTPMailer struct {
	Host, Port, Username, Password, From string
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(m.Host+":"+m.Port, auth, m.From, []string{to}, []byte(msg))
}

// New picks the SMTP mailer when a host is configured and the log mailer otherwise.
func New(host, port, username, password, from string) Mailer {
	if host == "" {
		return LogMailer{}
	}
	return SMTPMailer{Host: host, Port: port, Username: username, Password: password, From: from}
}
