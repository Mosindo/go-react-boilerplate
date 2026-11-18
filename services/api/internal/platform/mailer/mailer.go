// Package mailer sends transactional e-mails (password reset codes).
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strconv"
	"strings"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPMailer delivers mail through an SMTP relay. net/smtp upgrades to TLS via
// STARTTLS whenever the server supports it, and refuses PLAIN auth without TLS.
type SMTPMailer struct {
	addr string
	auth smtp.Auth
	from string
}

func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	return &SMTPMailer{addr: host + ":" + strconv.Itoa(port), auth: auth, from: from}
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid mail header")
	}
	msg := "From: " + m.from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n"
	return smtp.SendMail(m.addr, m.auth, m.from, []string{to}, []byte(msg))
}

// DevMailer prints e-mails to the server log. It is only wired when
// APP_ENV=development so codes never leak to production logs.
type DevMailer struct{}

func (DevMailer) Send(_ context.Context, to, subject, body string) error {
	log.Printf(`{"event":"dev_mail","to":%q,"subject":%q,"body":%q}`, to, subject, body)
	return nil
}

// DisabledMailer is used in production when SMTP is not configured.
type DisabledMailer struct{}

func (DisabledMailer) Send(_ context.Context, _, subject, _ string) error {
	log.Printf(`{"event":"mail_disabled","subject":%q,"hint":"configure SMTP_HOST to deliver e-mails"}`, subject)
	return nil
}
