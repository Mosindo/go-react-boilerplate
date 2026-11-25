package chat

import (
	"strings"
	"testing"
)

func TestNormalizeBody(t *testing.T) {
	got, err := NormalizeBody("  hello\x07 there\n  ")
	if err != nil || got != "hello there" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	if _, err := NormalizeBody("   \n "); err != ErrEmptyMessage {
		t.Fatalf("expected ErrEmptyMessage, got %v", err)
	}
	if _, err := NormalizeBody(strings.Repeat("é", MaxMessageRunes+1)); err != ErrMessageTooLong {
		t.Fatalf("expected ErrMessageTooLong, got %v", err)
	}
	if _, err := NormalizeBody(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Fatalf("2000 runes must be accepted: %v", err)
	}
}
