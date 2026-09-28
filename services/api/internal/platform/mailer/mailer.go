// Package mailer sends transactional email through an interface so services
// stay testable.
package mailer

import (
	"context"
	"errors"
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

// Log writes messages to the process log. Development and test only: it prints
// message bodies (which contain reset codes).
type Log struct{}

func (Log) Send(_ context.Context, msg Message) error {
	log.Printf("[mail:log] to=%s subject=%q body=%q", msg.To, msg.Subject, msg.Body)
	return nil
}

// SMTP sends through net/smtp. STARTTLS is used automatically when the server
// advertises it (net/smtp.SendMail); implicit TLS on port 465 is not supported.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Timeout  time.Duration
}

func (s SMTP) Send(ctx context.Context, msg Message) error {
	if err := checkHeader(msg.To); err != nil {
		return err
	}
	if err := checkHeader(msg.Subject); err != nil {
		return err
	}
	if err := checkHeader(s.From); err != nil {
		return err
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	payload := "From: " + s.From + "\r\n" +
		"To: " + msg.To + "\r\n" +
		"Subject: " + msg.Subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		strings.ReplaceAll(msg.Body, "\n", "\r\n") + "\r\n"

	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(addr, auth, s.From, []string{msg.To}, []byte(payload)) }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return errors.New("smtp send timed out")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// checkHeader blocks header injection.
func checkHeader(v string) error {
	if strings.ContainsAny(v, "\r\n") {
		return fmt.Errorf("illegal characters in mail header")
	}
	return nil
}
