package chat

import (
	"strings"
	"testing"
)

func TestNormalizeBody(t *testing.T) {
	got, err := NormalizeBody("  hello\x00 world​\n  ")
	if err != nil || got != "hello world​" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "   ", "\x00\x01", strings.Repeat("a", 2001)} {
		if _, err := NormalizeBody(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
	if _, err := NormalizeBody(strings.Repeat("é", 2000)); err != nil {
		t.Errorf("2000 runes must be accepted: %v", err)
	}
}

func TestPreview(t *testing.T) {
	if got := preview("a\n\n b"); got != "a b" {
		t.Errorf("got %q", got)
	}
	if got := []rune(preview(strings.Repeat("é", 200))); len(got) != 80 {
		t.Errorf("preview length %d", len(got))
	}
}
