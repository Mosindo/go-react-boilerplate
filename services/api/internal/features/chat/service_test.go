package chat

import (
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	got, err := NormalizeContent("  salut\nça va ?  ")
	if err != nil || got != "salut\nça va ?" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeContent(" \n\t "); err == nil {
		t.Error("whitespace-only must be rejected")
	}
	if _, err := NormalizeContent(strings.Repeat("é", MaxContentRune)); err != nil {
		t.Errorf("2000 runes must be accepted: %v", err)
	}
	if _, err := NormalizeContent(strings.Repeat("é", MaxContentRune+1)); err == nil {
		t.Error("2001 runes must be rejected")
	}
	if _, err := NormalizeContent("bad\x00byte"); err == nil {
		t.Error("control characters must be rejected")
	}
	if _, err := NormalizeContent("bad\xffutf8"); err == nil {
		t.Error("invalid UTF-8 must be rejected")
	}
}
