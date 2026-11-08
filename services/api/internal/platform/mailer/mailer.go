// Package mailer sends transactional e-mail. Without SMTP configuration messages are
// written to the log, which keeps local development and tests self-contained.
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"sync"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"mail_dev","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

type SMTPMailer struct {
	Host, Port, Username, Password, From string
}

func (m SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	for _, v := range []string{to, subject, m.From} {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("mailer: header injection attempt")
		}
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	return smtp.SendMail(m.Host+":"+m.Port, auth, m.From, []string{to}, []byte(msg))
}

// Memory records messages; used by tests.
type Memory struct {
	mu   sync.Mutex
	Sent []Message
}

type Message struct{ To, Subject, Body string }

func (m *Memory) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent = append(m.Sent, Message{to, subject, body})
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
