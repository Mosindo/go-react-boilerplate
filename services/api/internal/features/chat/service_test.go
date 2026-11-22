package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	if got, err := NormalizeContent("  salut !  "); err != nil || got != "salut !" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	if _, err := NormalizeContent(" \n\t "); !errors.Is(err, ErrMessageContentNeeded) {
		t.Fatalf("blank messages must be refused, got %v", err)
	}
	if _, err := NormalizeContent(strings.Repeat("é", maxMessageRunes+1)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("long messages must be refused, got %v", err)
	}
	if _, err := NormalizeContent(strings.Repeat("é", maxMessageRunes)); err != nil {
		t.Fatalf("limit is inclusive and counted in runes: %v", err)
	}
}
