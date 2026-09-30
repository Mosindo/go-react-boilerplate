// Package mailer sends transactional emails (password reset codes).
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

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPMailer delivers mail through an SMTP relay using STARTTLS/PLAIN auth.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, username: username, password: password, from: from}
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid mail header")
	}
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	addr := net.JoinHostPort(m.host, strconv.Itoa(m.port))
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}

// LogMailer is the development fallback: it prints emails to the server log
// instead of sending them. It is never used when APP_ENV=production.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}
