// Package mailer sends transactional emails (password reset codes).
package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"mime"
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

// sendTimeout bounds a whole SMTP exchange so a stalled relay cannot hang
// the request that triggered the email.
const sendTimeout = 20 * time.Second

func (m *SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid mail header")
	}
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		// RFC 2047: non-ASCII subjects must be encoded.
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
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

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return err
	}
	return m.deliver(conn, auth, to, []byte(msg))
}

// deliver mirrors smtp.SendMail on an already-dialled connection.
func (m *SMTPMailer) deliver(conn net.Conn, auth smtp.Auth, to string, msg []byte) error {
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(m.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// LogMailer is the development fallback: it prints emails to the server log
// instead of sending them. It is never used when APP_ENV=production.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}
