package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAgeOnHandlesBirthdayBoundary(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := map[string]int{
		"2008-06-15": 18, // birthday today: already 18
		"2008-06-16": 17, // turns 18 tomorrow
		"2000-01-01": 26,
		"1990-12-31": 35,
	}
	for birth, want := range cases {
		d, _ := time.Parse("2006-01-02", birth)
		if got := AgeOn(d, now); got != want {
			t.Errorf("AgeOn(%s) = %d, want %d", birth, got, want)
		}
	}
}

func TestParseBirthDateEnforcesAgeRules(t *testing.T) {
	s := &Service{now: func() time.Time { return time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) }}
	if _, err := s.parseBirthDate("2008-06-16"); !errors.Is(err, ErrUnderage) {
		t.Fatalf("17 year old must be refused, got %v", err)
	}
	if _, err := s.parseBirthDate("2008-06-15"); err != nil {
		t.Fatalf("18 year old must be accepted, got %v", err)
	}
	if _, err := s.parseBirthDate("1900-01-01"); err == nil {
		t.Fatal("implausible age must be refused")
	}
	if _, err := s.parseBirthDate("2000-02-30"); err == nil {
		t.Fatal("impossible date must be refused")
	}
}

func TestCleanText(t *testing.T) {
	if v, err := cleanText("f", "  Jean   Pierre ", 1, 40, false); err != nil || v != "Jean Pierre" {
		t.Fatalf("whitespace must be collapsed: %q %v", v, err)
	}
	if _, err := cleanText("f", "a\x00b", 0, 10, false); err == nil {
		t.Fatal("control characters must be refused")
	}
	if v, err := cleanText("bio", "line1\nline2", 0, 100, true); err != nil || v != "line1\nline2" {
		t.Fatalf("newlines allowed in multiline fields: %q %v", v, err)
	}
	if _, err := cleanText("f", strings.Repeat("é", 41), 1, 40, false); err == nil {
		t.Fatal("length is counted in characters, not bytes")
	}
	if _, err := cleanText("f", "   ", 1, 40, false); err == nil {
		t.Fatal("blank required field must be refused")
	}
}

func TestApproxDistanceRoundsUpToFiveKm(t *testing.T) {
	cases := map[float64]int{0: 5, 0.4: 5, 4.9: 5, 5: 10, 12.3: 15, 99.9: 100}
	for km, want := range cases {
		km := km
		if got := ApproxDistance(&km); got == nil || *got != want {
			t.Errorf("ApproxDistance(%v) = %v, want %d", km, got, want)
		}
	}
	if ApproxDistance(nil) != nil {
		t.Fatal("nil in, nil out")
	}
}
