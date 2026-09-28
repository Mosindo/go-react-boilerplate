package validate

import (
	"strings"
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestAgeOn(t *testing.T) {
	cases := []struct {
		birth, now string
		want       int
	}{
		{"2000-06-15", "2026-06-15", 26}, // birthday today
		{"2000-06-15", "2026-06-14", 25}, // day before
		{"2000-06-15", "2026-06-16", 26},
		{"2000-12-31", "2026-01-01", 25},
		{"2000-02-29", "2026-02-28", 25}, // leap-day birthday: older on Mar 1
		{"2000-02-29", "2026-03-01", 26},
		{"2000-02-29", "2028-02-29", 28},
		{"2026-06-15", "2026-06-15", 0},
	}
	for _, c := range cases {
		if got := AgeOn(d(c.birth), d(c.now)); got != c.want {
			t.Errorf("AgeOn(%s, %s) = %d, want %d", c.birth, c.now, got, c.want)
		}
	}
}

func TestIsAdultBoundary(t *testing.T) {
	now := d("2026-09-28")
	if !IsAdult(d("2008-09-28"), now) {
		t.Error("exactly 18 today must be an adult")
	}
	if IsAdult(d("2008-09-29"), now) {
		t.Error("turning 18 tomorrow must be underage")
	}
	if IsAdult(d("2020-01-01"), now) {
		t.Error("child")
	}
	// time of day must not matter
	if !IsAdult(time.Date(2008, 9, 28, 23, 59, 0, 0, time.UTC), time.Date(2026, 9, 28, 0, 0, 1, 0, time.UTC)) {
		t.Error("age is date based")
	}
}

func TestEmail(t *testing.T) {
	good := map[string]string{
		"  Bob@Example.COM ":    "bob@example.com",
		"a.b+tag@sub.domain.io": "a.b+tag@sub.domain.io",
	}
	for in, want := range good {
		got, err := Email(in)
		if err != nil || got != want {
			t.Errorf("Email(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "   ", "nope", "a@b", "@b.co", "a@", "Bob <bob@x.io>", "a b@x.io", "a@x.io, c@y.io", strings.Repeat("a", 250) + "@x.io", "a@x.io\r\nBcc: z@z.io"} {
		if _, err := Email(bad); err == nil {
			t.Errorf("Email(%q) should fail", bad)
		}
	}
}

func TestPassword(t *testing.T) {
	for _, ok := range []string{"12345678", strings.Repeat("a", 128), "pässwörd-ü"} {
		if err := Password(ok); err != nil {
			t.Errorf("Password(%q): %v", ok, err)
		}
	}
	for _, bad := range []string{"", "1234567", strings.Repeat("a", 129), "\xff\xfe\xfd\xfc\xfb\xfa\xf9\xf8"} {
		if err := Password(bad); err == nil {
			t.Errorf("Password(%q) should fail", bad)
		}
	}
}

func TestText(t *testing.T) {
	if got, err := Text("  hi  ", 1, 5, false); err != nil || got != "hi" {
		t.Errorf("trim: %q %v", got, err)
	}
	if _, err := Text("   ", 1, 5, false); err == nil {
		t.Error("blank must fail min length")
	}
	if got, err := Text("", 0, 5, true); err != nil || got != "" {
		t.Errorf("empty allowed with min 0: %q %v", got, err)
	}
	if _, err := Text("a\nb", 1, 5, false); err == nil {
		t.Error("newline in single-line text")
	}
	if got, err := Text("a\nb\tc", 1, 10, true); err != nil || got != "a\nb\tc" {
		t.Errorf("multiline: %q %v", got, err)
	}
	for _, bad := range []string{"a\x00b", "a\x1bb", "a b", "\xff", "a\u007fb"} {
		if _, err := Text(bad, 1, 10, true); err == nil {
			t.Errorf("Text(%q) should fail", bad)
		}
	}
	if _, err := Text("ééééé", 1, 4, false); err == nil {
		t.Error("length counts runes: 5 > 4")
	}
	if _, err := Text("éééé", 1, 4, false); err != nil {
		t.Error("length counts runes: 4 <= 4")
	}
}

func TestParseBirthDate(t *testing.T) {
	if _, err := ParseBirthDate(" 1990-02-03 "); err != nil {
		t.Error(err)
	}
	for _, bad := range []string{"", "1990-13-01", "1990-02-30", "03/02/1990", "1990-2-3x"} {
		if _, err := ParseBirthDate(bad); err == nil {
			t.Errorf("ParseBirthDate(%q) should fail", bad)
		}
	}
}
