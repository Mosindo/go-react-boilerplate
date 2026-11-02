// Package mail sends transactional emails (account recovery only).
package mail

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"example.com/api/internal/platform/config"
)

// Mailer implements auth.Mailer.
type Mailer struct {
	cfg     config.SMTPConfig
	logOnly bool
}

// New returns an SMTP mailer, or a log-only mailer when SMTP is not configured
// (development only: the reset token is written to the server log).
func New(cfg config.Config) *Mailer {
	return &Mailer{cfg: cfg.SMTP, logOnly: !cfg.SMTP.Enabled()}
}

func (m *Mailer) SendPasswordReset(_ context.Context, toEmail, token string) error {
	if m.logOnly {
		log.Printf(`{"event":"password_reset_dev","to":%q,"token":%q}`, toEmail, token)
		return nil
	}
	if strings.ContainsAny(toEmail, "\r\n") {
		return fmt.Errorf("invalid recipient")
	}
	body := "Subject: Réinitialisation de votre mot de passe\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n" +
		"From: " + m.cfg.From + "\r\nTo: " + toEmail + "\r\n\r\n" +
		"Voici votre code de réinitialisation (valable 1 heure) :\r\n\r\n" + token + "\r\n\r\n" +
		"Saisissez ce code dans l'application, écran « Mot de passe oublié ».\r\n\r\n" +
		"Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.\r\n"
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	return smtp.SendMail(m.cfg.Host+":"+m.cfg.Port, auth, m.cfg.From, []string{toEmail}, []byte(body))
}
