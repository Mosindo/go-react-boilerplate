// Package mailer sends transactional email. Without SMTP configuration it logs messages (dev only).
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

type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"mail_dev","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

type SMTPMailer struct{ Host, Port, User, Pass, From string }

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject+m.From, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Pass, m.Host)
	}
	return smtp.SendMail(m.Host+":"+m.Port, auth, m.From, []string{to}, []byte(msg))
}

func New(host, port, user, pass, from string) Mailer {
	if host == "" {
		return LogMailer{}
	}
	return SMTPMailer{Host: host, Port: port, User: user, Pass: pass, From: from}
}
