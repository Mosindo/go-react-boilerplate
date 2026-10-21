// Package mail sends transactional e-mails (password recovery).
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

// LogMailer is the development fallback used when no SMTP server is configured.
type LogMailer struct{}

func (LogMailer) Send(to, subject, body string) error {
	log.Printf("[dev mailer] SMTP not configured, mail to=%s subject=%q\n%s", to, subject, body)
	return nil
}

type SMTPMailer struct {
	Host, User, Password, From string
	Port                       int
}

func (m SMTPMailer) Send(to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("mail: header injection attempt")
	}
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Password, m.Host)
	}
	return smtp.SendMail(addr, auth, m.From, []string{to}, []byte(msg))
}
