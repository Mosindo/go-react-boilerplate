package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeBody(t *testing.T) {
	got, err := NormalizeBody("  salut\u0000 !\n ")
	if err != nil || got != "salut !" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	if _, err := NormalizeBody(" \u0007 "); !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected empty error, got %v", err)
	}
	if _, err := NormalizeBody(strings.Repeat("é", MaxMessageLength+1)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("expected too long error, got %v", err)
	}
	if _, err := NormalizeBody(strings.Repeat("é", MaxMessageLength)); err != nil {
		t.Fatalf("2000 runes must be accepted: %v", err)
	}
}
