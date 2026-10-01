package profiles

import (
	"strings"
	"testing"
	"time"
)

func TestAgeOnHandlesBirthdayBoundary(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	for birth, want := range map[string]int{
		"2008-06-15": 18, // birthday today
		"2008-06-16": 17, // tomorrow
		"2008-06-14": 18,
		"1990-12-31": 35,
		"2000-02-29": 26,
	} {
		b, _ := time.Parse("2006-01-02", birth)
		if got := AgeOn(b, now); got != want {
			t.Errorf("AgeOn(%s) = %d, want %d", birth, got, want)
		}
	}
}

func TestRoundCoordinateKeepsAboutOneKilometre(t *testing.T) {
	for in, want := range map[float64]float64{45.764043: 45.76, 4.835659: 4.84, -73.9857: -73.99, 0.004: 0} {
		if got := RoundCoordinate(in); got != want {
			t.Errorf("RoundCoordinate(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestCleanText(t *testing.T) {
	got, err := cleanText("  Jean \t\n  Pierre\x00 ", 50, false)
	if err != nil || got != "Jean Pierre" {
		t.Fatalf("single line: %q %v", got, err)
	}
	got, err = cleanText("line1\r\nline2\x07", 50, true)
	if err != nil || got != "line1\nline2 " && got != "line1\nline2" {
		t.Fatalf("multiline: %q %v", got, err)
	}
	if _, err := cleanText(strings.Repeat("é", 11), 10, false); err == nil {
		t.Fatal("limit counts runes, not bytes: 11 runes must fail a 10 limit")
	}
	if _, err := cleanText(strings.Repeat("é", 10), 10, false); err != nil {
		t.Fatalf("10 runes must pass a 10 limit: %v", err)
	}
}

func TestUniqueSorted(t *testing.T) {
	got := uniqueSorted([]string{" b", "a", "b", "", "a "})
	if strings.Join(got, ",") != "a,b" {
		t.Fatalf("unexpected %v", got)
	}
}
