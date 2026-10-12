package auth

import (
	"context"
	"log"
	"os"
	"strings"
)

// LogMailer is the development mailer. By default it only logs that a reset was
// requested. With MAIL_LOG_TOKENS=true (development only) it also logs the reset
// token so the flow can be exercised without an SMTP server. Production
// deployments must plug a real Mailer via Service.WithMailer.
type LogMailer struct{}

func (LogMailer) SendPasswordReset(_ context.Context, email, token string) error {
	if strings.EqualFold(os.Getenv("MAIL_LOG_TOKENS"), "true") {
		log.Printf(`{"event":"password_reset_requested","dev_only":true,"email":%q,"token":%q}`, email, token)
		return nil
	}
	log.Printf(`{"event":"password_reset_requested","note":"no mailer configured","email_domain":%q}`, emailDomain(email))
	return nil
}

func emailDomain(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		return email[i+1:]
	}
	return ""
}
