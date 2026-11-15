package chat

import (
	"strings"
	"testing"
)

func TestValidateMessage(t *testing.T) {
	if _, err := ValidateMessage(" \n\t "); err != ErrEmptyMessage {
		t.Fatalf("expected empty error, got %v", err)
	}
	if _, err := ValidateMessage(strings.Repeat("é", MaxMessageRunes+1)); err != ErrMessageTooLong {
		t.Fatalf("expected too long, got %v", err)
	}
	got, err := ValidateMessage("  salut‮ \n\n\n\n ça va ?  ")
	if err != nil || got != "salut\n\nça va ?" {
		t.Fatalf("unexpected %q %v", got, err)
	}
}

func TestPreview(t *testing.T) {
	if p := preview(strings.Repeat("a", 200)); len([]rune(p)) != 118 {
		t.Fatalf("unexpected preview length %d", len([]rune(p)))
	}
}
