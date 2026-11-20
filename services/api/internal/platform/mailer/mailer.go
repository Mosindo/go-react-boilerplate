// Package mailer sends transactional e-mails (account recovery).
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

// SMTPMailer delivers e-mails through an SMTP relay (STARTTLS when offered).
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (m *SMTPMailer) Send(_ context.Context, msg Message) error {
	if strings.ContainsAny(msg.To, "\r\n") || strings.ContainsAny(msg.Subject, "\r\n") {
		return fmt.Errorf("mailer: header injection attempt")
	}
	addr := net.JoinHostPort(m.Host, strconv.Itoa(m.Port))
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
	return smtp.SendMail(addr, auth, m.From, []string{msg.To}, []byte(payload))
}

// DevMailer is used when no SMTP relay is configured. Outside production it
// prints the e-mail to the API logs so the recovery flow can be exercised
// locally; in production it refuses to leak the content and returns an error.
type DevMailer struct {
	Production bool
}

func (m *DevMailer) Send(_ context.Context, msg Message) error {
	if m.Production {
		log.Printf(`{"event":"mail_not_sent","reason":"smtp_not_configured","subject":%q}`, msg.Subject)
		return fmt.Errorf("mailer: SMTP is not configured")
	}
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, msg.To, msg.Subject, msg.Body)
	return nil
}
