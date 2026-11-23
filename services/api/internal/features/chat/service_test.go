package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	got, err := NormalizeContent("  salut\x00 toi\n ")
	if err != nil || got != "salut toi" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeContent(" \n\t "); !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected empty error, got %v", err)
	}
	if _, err := NormalizeContent(strings.Repeat("é", MaxMessageRunes+1)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("expected too long error, got %v", err)
	}
	if _, err := NormalizeContent(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Fatalf("2000 runes must be accepted: %v", err)
	}
}
