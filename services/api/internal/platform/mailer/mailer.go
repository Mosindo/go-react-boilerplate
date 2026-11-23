// Package mailer sends transactional emails (account recovery).
package mailer

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net"
	"net/mail"
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

// SMTP delivers mail through an authenticated SMTP relay (STARTTLS when the
// server advertises it, which net/smtp does automatically).
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
	from, err := mail.ParseAddress(m.From)
	if err != nil {
		return fmt.Errorf("invalid SMTP_FROM: %w", err)
	}
	to, err := mail.ParseAddress(msg.To)
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}
	addr := net.JoinHostPort(m.Host, strconv.Itoa(m.Port))
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	payload := strings.Join([]string{
		"From: " + from.String(),
		"To: " + to.Address,
		"Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject),
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		msg.Body,
	}, "\r\n")
	return smtp.SendMail(addr, auth, from.Address, []string{to.Address}, []byte(payload))
}

// DevLog prints emails to the API log. It is only wired outside production
// (config refuses to start in production without SMTP).
type DevLog struct{}

func (DevLog) Send(_ context.Context, msg Message) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, msg.To, msg.Subject, msg.Body)
	return nil
}
