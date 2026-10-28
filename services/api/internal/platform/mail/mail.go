// Package mail sends transactional email (password recovery).
package mail

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(to, subject, body string) error
}

// LogMailer prints messages to the server log. Development only: it is rejected in production by config.
type LogMailer struct{}

func (LogMailer) Send(to, subject, body string) error {
	log.Printf("[dev-mail] to=%s subject=%q\n%s", to, subject, body)
	return nil
}

type SMTPMailer struct {
	Host, Port, Username, Password, From string
}

func (m SMTPMailer) Send(to, subject, body string) error {
	if strings.ContainsAny(to+subject+m.From, "\r\n") {
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
