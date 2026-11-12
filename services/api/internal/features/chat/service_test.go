package chat

import (
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	if got, err := NormalizeContent("  hello\r\nworld  "); err != nil || got != "hello\nworld" {
		t.Fatalf("unexpected: %q %v", got, err)
	}
	if _, err := NormalizeContent(" \n "); err != ErrMessageContentNeeded {
		t.Fatal("blank message must be rejected")
	}
	if _, err := NormalizeContent(strings.Repeat("é", 2001)); err != ErrMessageTooLong {
		t.Fatal("2001 runes must be rejected")
	}
	if _, err := NormalizeContent(strings.Repeat("é", 2000)); err != nil {
		t.Fatal("2000 runes must be accepted")
	}
}
