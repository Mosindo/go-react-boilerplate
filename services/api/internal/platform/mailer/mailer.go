// Package mailer sends transactional email (password reset). Without SMTP configuration it
// logs messages, which is only allowed outside production (see config.Load).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	log.Printf(`{"event":"mail_dev","to":%q,"subject":%q,"body":%q}`, msg.To, msg.Subject, msg.Body)
	return nil
}

type SMTPMailer struct {
	Host, Port, Username, Password, From string
}

func (m SMTPMailer) Send(_ context.Context, msg Message) error {
	if strings.ContainsAny(msg.To+msg.Subject, "\r\n") {
		return fmt.Errorf("mailer: header injection attempt")
	}
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	raw := "From: " + m.From + "\r\nTo: " + msg.To + "\r\nSubject: " + msg.Subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + msg.Body
	return smtp.SendMail(m.Host+":"+m.Port, auth, m.From, []string{msg.To}, []byte(raw))
}
