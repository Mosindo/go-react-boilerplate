package chat

import (
	"strings"
	"testing"
	"time"
)

func TestCleanBody(t *testing.T) {
	if got, err := CleanBody("  bonjour\nça va ? 😀 "); err != nil || got != "bonjour\nça va ? 😀" {
		t.Fatalf("got %q %v", got, err)
	}
	for _, bad := range []string{"", " \n\t ", strings.Repeat("é", MaxBodyRunes+1), "a\x00b", "a\x1bb", "\xff\xfe"} {
		if _, err := CleanBody(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
	if _, err := CleanBody(strings.Repeat("é", MaxBodyRunes)); err != nil {
		t.Errorf("exactly max runes must pass: %v", err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 678000, time.UTC)
	c := encodeCursor(ts, "abc")
	got, id, ok := decodeCursor(c)
	if !ok || !got.Equal(ts) || id != "abc" {
		t.Fatalf("round trip failed: %v %v %v", got, id, ok)
	}
	for _, bad := range []string{"", "!!!", "bm90LWEtY3Vyc29y"} {
		if _, _, ok := decodeCursor(bad); ok {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
