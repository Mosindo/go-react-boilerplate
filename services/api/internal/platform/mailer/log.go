package mailer

import (
	"context"
	"log"
)

// LogMailer writes emails to the process log. Development only: it exposes the
// message body, which contains the password reset code.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	log.Printf(`{"event":"mail_dev","to":%q,"subject":%q,"body":%q}`, msg.To, msg.Subject, msg.Body)
	return nil
}

// DiscardMailer drops every message (production default until a real mailer is wired).
type DiscardMailer struct{}

func (DiscardMailer) Send(context.Context, Message) error { return nil }
