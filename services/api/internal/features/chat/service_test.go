package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestCleanBody(t *testing.T) {
	if got, err := CleanBody("  salut \n"); err != nil || got != "salut" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := CleanBody("   \n\t"); !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected empty error, got %v", err)
	}
	if _, err := CleanBody(strings.Repeat("é", 2001)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("expected too long, got %v", err)
	}
	if _, err := CleanBody(strings.Repeat("é", 2000)); err != nil {
		t.Fatalf("2000 runes must pass: %v", err)
	}
	if _, err := CleanBody("bad\x00byte"); err == nil {
		t.Fatal("control characters must be rejected")
	}
}
