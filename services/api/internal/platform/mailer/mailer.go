// Package mailer envoie les emails transactionnels (recuperation de compte).
package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

// Mailer envoie un email texte simple.
type Mailer interface {
	Send(to, subject, body string) error
}

// Log affiche les emails dans les logs. Reserve au developpement: le contenu
// (contenant le lien de reinitialisation) est ecrit en clair.
type Log struct{}

func (Log) Send(to, subject, body string) error {
	log.Printf("[mailer:dev] to=%s subject=%q\n%s", to, subject, body)
	return nil
}

// SMTP envoie via un serveur SMTP (STARTTLS gere par net/smtp si annonce).
type SMTP struct {
	Host, Port, User, Password, From string
}

func (s SMTP) Send(to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "From: " + s.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Password, s.Host)
	}
	return smtp.SendMail(s.Host+":"+s.Port, auth, s.From, []string{to}, []byte(msg))
}
