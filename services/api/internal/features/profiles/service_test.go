package profiles

import (
	"strings"
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	today := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-03-10", 18}, // birthday today
		{"2008-03-11", 17}, // birthday tomorrow: still 17
		{"2000-12-31", 25},
		{"1990-02-28", 36},
	}
	for _, tc := range cases {
		b, _ := time.Parse("2006-01-02", tc.birth)
		if got := AgeOn(b, today); got != tc.want {
			t.Fatalf("%s: want %d got %d", tc.birth, tc.want, got)
		}
	}
}

func TestCleanText(t *testing.T) {
	if _, err := cleanText("   ", 10, "x", true); err == nil {
		t.Fatal("blank required text must fail")
	}
	if v, err := cleanText("  hello ", 10, "x", true); err != nil || v != "hello" {
		t.Fatalf("trim failed: %q %v", v, err)
	}
	if _, err := cleanText(strings.Repeat("é", 11), 10, "x", false); err == nil {
		t.Fatal("must count runes, not bytes")
	}
	if _, err := cleanText("bad\x00text", 10, "x", false); err == nil {
		t.Fatal("control characters must fail")
	}
}

func TestRound2HidesExactPosition(t *testing.T) {
	if got := round2(48.856614); got != 48.86 {
		t.Fatalf("got %v", got)
	}
}

func TestDedupe(t *testing.T) {
	got := dedupe([]string{" Man", "man", "WOMAN", ""})
	if len(got) != 2 || got[0] != "man" || got[1] != "woman" {
		t.Fatalf("got %v", got)
	}
}
