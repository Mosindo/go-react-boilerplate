package matches

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeMessage(t *testing.T) {
	if got, err := NormalizeMessage("  salut \x00 ! "); err != nil || got != "salut  !" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeMessage(" \n\t "); !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected empty error, got %v", err)
	}
	if _, err := NormalizeMessage(strings.Repeat("é", 2001)); !errors.Is(err, ErrMessageLong) {
		t.Fatalf("expected too long error, got %v", err)
	}
	if _, err := NormalizeMessage(strings.Repeat("é", 2000)); err != nil {
		t.Fatalf("2000 runes must be accepted: %v", err)
	}
}
