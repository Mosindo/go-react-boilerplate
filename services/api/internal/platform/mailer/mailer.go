// Package mailer sends transactional emails (password recovery codes).
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

// SMTP sends mail through an SMTP relay using STARTTLS when the server offers it.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (m SMTP) Send(_ context.Context, msg Message) error {
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
	return smtp.SendMail(addr, auth, m.From, []string{msg.To}, []byte(body))
}

// DevLog prints emails to the API log. It is only wired when APP_ENV is
// "development" or "test" so recovery codes are never logged in production.
type DevLog struct{}

func (DevLog) Send(_ context.Context, msg Message) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, msg.To, msg.Subject, msg.Body)
	return nil
}

// Disabled is used in production when SMTP is not configured.
type Disabled struct{}

func (Disabled) Send(_ context.Context, _ Message) error {
	return fmt.Errorf("mailer not configured: set SMTP_HOST and SMTP_FROM")
}
