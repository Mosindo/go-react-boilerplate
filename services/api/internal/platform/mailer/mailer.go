// Package mailer sends transactional email. Without SMTP configuration the LogMailer
// prints messages to the server log, which is only acceptable in development.
package mailer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

type SMTPMailer struct {
	Host, Port, Username, Password, From string
}

func (m SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	for _, v := range []string{to, subject, m.From} {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("mailer: header injection attempt")
		}
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(net.JoinHostPort(m.Host, m.Port), auth, m.From, []string{to}, []byte(msg))
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
		return fmt.Errorf("mailer: smtp timeout")
	}
}
