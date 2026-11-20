package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeMessageBody(t *testing.T) {
	got, err := NormalizeMessageBody("  Salut\x00 toi !\n\tÇa va ?  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Salut toi !\n\tÇa va ?" {
		t.Fatalf("unexpected normalization %q", got)
	}
	if _, err := NormalizeMessageBody(" \n\t "); !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected ErrEmptyMessage, got %v", err)
	}
	if _, err := NormalizeMessageBody(strings.Repeat("é", 2001)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("expected ErrMessageTooLong, got %v", err)
	}
	if _, err := NormalizeMessageBody(strings.Repeat("é", 2000)); err != nil {
		t.Fatalf("2000 runes must be accepted: %v", err)
	}
}
