package chat

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeBody(t *testing.T) {
	got, err := NormalizeBody("  salut\x00 toi \n")
	if err != nil || got != "salut toi" {
		t.Fatalf("NormalizeBody = %q, %v", got, err)
	}
	if _, err := NormalizeBody("   \n\t "); err != ErrInvalidMessage {
		t.Fatal("blank messages must be rejected")
	}
	if _, err := NormalizeBody(strings.Repeat("é", 2001)); err != ErrInvalidMessage {
		t.Fatal("long messages must be rejected")
	}
	if _, err := NormalizeBody(strings.Repeat("é", 2000)); err != nil {
		t.Fatal("2000 characters must be accepted")
	}
}

func TestToMessageReadState(t *testing.T) {
	at := time.Now()
	before, after := at.Add(-time.Second), at.Add(time.Second)
	m := StoredMessage{ID: "m", SenderID: "me", CreatedAt: at}
	if toMessage(m, "me", &before).IsRead {
		t.Fatal("message newer than partner read marker is unread")
	}
	if !toMessage(m, "me", &after).IsRead {
		t.Fatal("message older than partner read marker is read")
	}
	if got := toMessage(m, "partner", &after); got.IsMine || got.IsRead {
		t.Fatal("received messages carry no read receipt")
	}
}
