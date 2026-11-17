// Package mailer delivers transactional e-mails (password reset codes).
package mailer

import (
	"context"
	"fmt"
	"log"
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

// SMTPMailer sends plain-text mail through an SMTP relay using STARTTLS when offered.
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (m *SMTPMailer) Send(_ context.Context, msg Message) error {
	if strings.ContainsAny(msg.To, "\r\n") || strings.ContainsAny(msg.Subject, "\r\n") {
		return fmt.Errorf("invalid mail header")
	}
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	payload := strings.Join([]string{
		"From: " + m.From,
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		msg.Body,
	}, "\r\n")
	return smtp.SendMail(m.Host+":"+strconv.Itoa(m.Port), auth, m.From, []string{msg.To}, []byte(payload))
}

// LogMailer is the development fallback: it writes the message to the server log so
// password reset can be exercised locally. Config forbids it in production.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, msg.To, msg.Subject, msg.Body)
	return nil
}
