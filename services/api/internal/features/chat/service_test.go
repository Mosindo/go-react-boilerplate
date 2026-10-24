package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	got, err := normalizeContent("  hello\x00 world\n  ")
	if err != nil || got != "hello world" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := normalizeContent(" \n\t "); !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("blank must be rejected: %v", err)
	}
	if _, err := normalizeContent(strings.Repeat("é", MaxMessageRunes+1)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("length is counted in runes: %v", err)
	}
	if _, err := normalizeContent(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Fatalf("exactly the max is fine: %v", err)
	}
}
