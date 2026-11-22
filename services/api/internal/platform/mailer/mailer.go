// Package mailer sends transactional e-mails (account recovery).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"sync"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// LogMailer prints e-mails to the server log. It is meant for local
// development only; config.Load refuses it when APP_ENV=production.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, msg.To, msg.Subject, msg.Body)
	return nil
}

// SMTPMailer delivers through an SMTP relay using STARTTLS-capable auth.
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (m SMTPMailer) Send(_ context.Context, msg Message) error {
	if strings.ContainsAny(msg.To, "\r\n") || strings.ContainsAny(msg.Subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	body := "From: " + m.From + "\r\n" +
		"To: " + msg.To + "\r\n" +
		"Subject: " + msg.Subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		msg.Body + "\r\n"
	return smtp.SendMail(net.JoinHostPort(m.Host, m.Port), auth, m.From, []string{msg.To}, []byte(body))
}

// Memory records messages; used by tests.
type Memory struct {
	mu   sync.Mutex
	Sent []Message
}

func (m *Memory) Send(_ context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent = append(m.Sent, msg)
	return nil
}

func (m *Memory) Last() (Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Sent) == 0 {
		return Message{}, false
	}
	return m.Sent[len(m.Sent)-1], true
}
