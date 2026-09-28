package mailer

import (
	"context"
	"strings"
	"testing"
)

func TestSMTPRefusesHeaderInjection(t *testing.T) {
	s := SMTP{Host: "127.0.0.1", Port: 1, From: "noreply@x.invalid"}
	for _, m := range []Message{
		{To: "a@x.invalid\r\nBcc: evil@x.invalid", Subject: "s", Body: "b"},
		{To: "a@x.invalid", Subject: "s\nBcc: evil@x.invalid", Body: "b"},
	} {
		err := s.Send(context.Background(), m)
		if err == nil || !strings.Contains(err.Error(), "illegal") {
			t.Errorf("expected header injection to be refused before any network access, got %v", err)
		}
	}
	bad := SMTP{Host: "127.0.0.1", Port: 1, From: "a@x.invalid\r\nX: y"}
	if err := bad.Send(context.Background(), Message{To: "a@x.invalid", Subject: "s", Body: "b"}); err == nil {
		t.Error("From injection must be refused")
	}
}

func TestLogMailerDoesNotFail(t *testing.T) {
	if err := (Log{}).Send(context.Background(), Message{To: "a@x.invalid", Subject: "s", Body: "b"}); err != nil {
		t.Fatal(err)
	}
}
