package chat

import (
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	got, err := normalizeContent("  hello\x00 world​ \n")
	if err != nil || got != "hello world​" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := normalizeContent(" \n\t "); err != ErrEmptyMessage {
		t.Fatalf("blank: %v", err)
	}
	if _, err := normalizeContent(strings.Repeat("😀", MaxMessageRunes+1)); err != ErrMessageTooLong {
		t.Fatalf("too long: %v", err)
	}
	if _, err := normalizeContent(strings.Repeat("😀", MaxMessageRunes)); err != nil {
		t.Fatalf("limit must be counted in characters, not bytes: %v", err)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("abcdef", 4); got != "abc…" {
		t.Fatal(got)
	}
	if got := truncateRunes("abc", 4); got != "abc" {
		t.Fatal(got)
	}
}
