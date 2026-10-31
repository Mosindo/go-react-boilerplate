package validate

import (
	"testing"
	"time"
)

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18},
		{"2008-06-16", 17},
		{"1990-01-01", 36},
	}
	for _, tc := range cases {
		b, _ := time.Parse("2006-01-02", tc.birth)
		if got := AgeAt(b, now); got != tc.want {
			t.Errorf("AgeAt(%s)=%d want %d", tc.birth, got, tc.want)
		}
	}
}

func TestParseBirthDate(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	if _, err := ParseBirthDate("2008-06-15", now); err != nil {
		t.Fatalf("18th birthday must be accepted: %v", err)
	}
	if _, err := ParseBirthDate("2008-06-16", now); err != ErrUnderage {
		t.Fatalf("17 must be rejected, got %v", err)
	}
	if _, err := ParseBirthDate("2027-01-01", now); err != ErrInvalidBirthDate {
		t.Fatalf("future date must be invalid, got %v", err)
	}
	if _, err := ParseBirthDate("1800-01-01", now); err != ErrInvalidBirthDate {
		t.Fatalf("too old must be invalid, got %v", err)
	}
	if _, err := ParseBirthDate("not-a-date", now); err != ErrInvalidBirthDate {
		t.Fatalf("garbage must be invalid, got %v", err)
	}
}

func TestCleanText(t *testing.T) {
	got, ok := CleanText("  hello\x00 world \n", 50)
	if !ok || got != "hello world" {
		t.Fatalf("unexpected %q %v", got, ok)
	}
	if _, ok := CleanText("abcdef", 3); ok {
		t.Fatal("expected length rejection")
	}
}
