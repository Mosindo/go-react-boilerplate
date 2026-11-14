package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeMessage(t *testing.T) {
	got, err := NormalizeMessage("  Salut !\u0000\n ça va ?  ")
	if err != nil || got != "Salut !\n ça va ?" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	for _, bad := range []string{"", "   ", "\u0000\u0001", strings.Repeat("a", 2001)} {
		if _, err := NormalizeMessage(bad); !errors.Is(err, ErrInvalidMessage) {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
	if _, err := NormalizeMessage(strings.Repeat("é", 2000)); err != nil {
		t.Fatalf("2000 runes must be accepted: %v", err)
	}
}
