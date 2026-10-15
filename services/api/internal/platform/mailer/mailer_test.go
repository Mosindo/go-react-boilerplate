package mailer

import (
	"context"
	"testing"
)

func TestSMTPRequiresHostAndFrom(t *testing.T) {
	if _, err := NewSMTP(SMTPConfig{Host: "", From: "a@b.test"}); err == nil {
		t.Fatal("missing host must be rejected")
	}
	if _, err := NewSMTP(SMTPConfig{Host: "smtp.test", From: ""}); err == nil {
		t.Fatal("missing from address must be rejected")
	}
	if m, err := NewSMTP(SMTPConfig{Host: "smtp.test", From: "a@b.test"}); err != nil || m.cfg.Port != "587" {
		t.Fatalf("expected default port 587: %v", err)
	}
}

func TestSMTPRejectsHeaderInjection(t *testing.T) {
	m, _ := NewSMTP(SMTPConfig{Host: "127.0.0.1", Port: "1", From: "a@b.test"})
	if err := m.Send(context.Background(), "victim@x.test\r\nBcc: spy@x.test", "Hi", "body"); err == nil {
		t.Fatal("CRLF in the recipient must be rejected before any connection")
	}
	if err := m.Send(context.Background(), "victim@x.test", "Hi\r\nBcc: spy@x.test", "body"); err == nil {
		t.Fatal("CRLF in the subject must be rejected before any connection")
	}
}
