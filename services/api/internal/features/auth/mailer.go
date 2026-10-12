package auth

import (
	"context"
	"log"
)

// LogMailer is the development mailer: it only logs that a reset was requested.
// The token itself is never logged; wire a real Mailer in production.
type LogMailer struct{}

func (LogMailer) SendPasswordReset(_ context.Context, email, _ string) error {
	log.Printf(`{"event":"password_reset_requested","note":"no mailer configured","email_domain":%q}`, emailDomain(email))
	return nil
}

func emailDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			return email[i+1:]
		}
	}
	return ""
}
