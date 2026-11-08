package validate

import (
	"testing"
	"time"
)

func TestAge(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"1990-01-01", 36},
		{"2000-02-29", 26},
	}
	for _, tc := range cases {
		b, err := ParseBirthDate(tc.birth)
		if err != nil {
			t.Fatal(err)
		}
		if got := Age(b, now); got != tc.want {
			t.Errorf("Age(%s) = %d, want %d", tc.birth, got, tc.want)
		}
	}
}

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "first.last@example.com"} {
		if !ValidEmail(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "a@b", "Name <a@b.co>", "no-at.com", "a@@b.co"} {
		if ValidEmail(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestCleanText(t *testing.T) {
	if _, ok := CleanText("hello\x00", 10, false); ok {
		t.Error("control chars must be rejected")
	}
	if got, ok := CleanText("  hi\nthere ", 20, true); !ok || got != "hi\nthere" {
		t.Errorf("got %q %v", got, ok)
	}
	if _, ok := CleanText("héllo wörld", 5, false); ok {
		t.Error("too long must be rejected by rune count")
	}
}
