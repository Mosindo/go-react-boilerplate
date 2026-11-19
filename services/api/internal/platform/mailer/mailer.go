// Package mailer sends transactional emails (account recovery). SMTP is used when
// configured; otherwise, outside production, messages are written to the log so the
// recovery flow stays testable locally.
package mailer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// LogMailer prints messages to the server log. Never used in production (config enforces SMTP).
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	log.Printf(`{"event":"mail_dev_delivery","to":%q,"subject":%q,"body":%q}`, msg.To, msg.Subject, msg.Body)
	return nil
}

type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (m SMTPMailer) Send(ctx context.Context, msg Message) error {
	if strings.ContainsAny(msg.To, "\r\n") || strings.ContainsAny(msg.Subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	addr := net.JoinHostPort(m.Host, strconv.Itoa(m.Port))
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	body := strings.Join([]string{
		"From: " + m.From,
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		msg.Body,
	}, "\r\n")

	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(addr, auth, m.From, []string{msg.To}, []byte(body)) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
