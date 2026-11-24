package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeMessage(t *testing.T) {
	got, err := NormalizeMessage("  salut\x07 !\n ")
	if err != nil || got != "salut !" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	if _, err := NormalizeMessage(" \n\t "); !errors.Is(err, ErrEmptyMessage) {
		t.Fatal("expected empty message error")
	}
	if _, err := NormalizeMessage(strings.Repeat("é", MaxMessageRunes+1)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatal("expected too long error")
	}
	if _, err := NormalizeMessage(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Fatal("2000 runes must be accepted")
	}
}
