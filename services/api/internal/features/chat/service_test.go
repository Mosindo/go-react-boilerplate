package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeMessage(t *testing.T) {
	got, err := NormalizeMessage("  salut\x07 toi\n ")
	if err != nil || got != "salut toi" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	if _, err := NormalizeMessage(" \n\t "); !errors.Is(err, ErrMessageEmpty) {
		t.Fatalf("expected empty error, got %v", err)
	}
	if _, err := NormalizeMessage(strings.Repeat("é", MaxMessageRunes+1)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("expected too long error, got %v", err)
	}
	if got, err := NormalizeMessage(strings.Repeat("é", MaxMessageRunes)); err != nil || len([]rune(got)) != MaxMessageRunes {
		t.Fatalf("max-length message must be accepted: %v", err)
	}
}
