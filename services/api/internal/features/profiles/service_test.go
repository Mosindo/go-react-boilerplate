package profiles

import (
	"testing"
	"time"
)

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"2008-12-31", 17},
		{"2000-02-29", 26},
		{"1990-01-01", 36},
	} {
		birth, _ := time.Parse("2006-01-02", tc.birth)
		if got := AgeAt(birth, now); got != tc.want {
			t.Errorf("AgeAt(%s) = %d, want %d", tc.birth, got, tc.want)
		}
	}
}

func TestCleanText(t *testing.T) {
	if _, err := cleanText("bad\x07text", 10, false); err == nil {
		t.Fatal("control characters must be rejected")
	}
	if got, err := cleanText("  ok\nline ", 10, true); err != nil || got != "ok\nline" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := cleanText("héééééééééé", 10, false); err == nil {
		t.Fatal("length counts runes, 11 runes must fail")
	}
	if _, err := cleanText("hééééééééé", 10, false); err != nil {
		t.Fatalf("10 runes must pass: %v", err)
	}
}
