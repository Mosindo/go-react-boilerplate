// Package mail sends transactional emails (password recovery).
package mail

import (
	"context"
	"errors"
	"log"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTP sends mail through an authenticated SMTP relay (STARTTLS is negotiated when offered).
type SMTP struct {
	Host, Port, Username, Password, From string
}

func (s SMTP) Send(_ context.Context, to, subject, body string) error {
	for _, v := range []string{to, subject, s.From} {
		if strings.ContainsAny(v, "\r\n") {
			return errors.New("mail: header injection attempt")
		}
	}
	msg := "From: " + s.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	return smtp.SendMail(s.Host+":"+s.Port, auth, s.From, []string{to}, []byte(msg))
}

// Log writes emails to the server log. Development only: it prints reset tokens.
type Log struct{}

func (Log) Send(_ context.Context, to, subject, body string) error {
	log.Printf("[dev mail] to=%s subject=%q\n%s", to, subject, body)
	return nil
}

// Capture stores sent emails in memory (tests).
type Capture struct {
	Sent []Message
}

type Message struct{ To, Subject, Body string }

func (c *Capture) Send(_ context.Context, to, subject, body string) error {
	c.Sent = append(c.Sent, Message{To: to, Subject: subject, Body: body})
	return nil
}
