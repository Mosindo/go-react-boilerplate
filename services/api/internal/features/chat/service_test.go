package chat

import (
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	if _, err := NormalizeContent("   "); err != ErrMessageContentNeeded {
		t.Fatalf("expected empty content error, got %v", err)
	}
	if _, err := NormalizeContent(strings.Repeat("é", MaxMessageRunes+1)); err != ErrMessageTooLong {
		t.Fatalf("expected too long error, got %v", err)
	}
	got, err := NormalizeContent("  salut\x00 toi\n:)  ")
	if err != nil || got != "salut toi\n:)" {
		t.Fatalf("unexpected normalisation %q %v", got, err)
	}
}
