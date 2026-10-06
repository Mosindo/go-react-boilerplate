package profiles

import (
	"errors"
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"1990-01-01", 36},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(b, now); got != c.want {
			t.Errorf("AgeOn(%s)=%d want %d", c.birth, got, c.want)
		}
	}
}

func TestParseBirthdateEnforcesAdultAccess(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	if _, err := parseBirthdate("2008-06-16", now); !errors.Is(err, ErrUnderage) {
		t.Fatalf("17-year-old must be rejected, got %v", err)
	}
	if _, err := parseBirthdate("2008-06-15", now); err != nil {
		t.Fatalf("18-year-old must be accepted, got %v", err)
	}
	for _, bad := range []string{"", "15/06/1990", "2030-01-01", "1900-01-01", "not-a-date"} {
		if _, err := parseBirthdate(bad, now); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestNormalizeFirstName(t *testing.T) {
	if got, err := normalizeFirstName("  Anne-Laure   "); err != nil || got != "Anne-Laure" {
		t.Fatalf("got %q %v", got, err)
	}
	for _, bad := range []string{"", "   ", "Bob<script>", "R2D2", "A\x00"} {
		if _, err := normalizeFirstName(bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestNormalizeBioStripsControlCharsAndLimitsLength(t *testing.T) {
	got, err := normalizeBio("  hello\x07 world\nline2  ")
	if err != nil || got != "hello world\nline2" {
		t.Fatalf("got %q %v", got, err)
	}
	long := make([]rune, MaxBioRunes+1)
	for i := range long {
		long[i] = 'é'
	}
	if _, err := normalizeBio(string(long)); !errors.Is(err, ErrBioTooLong) {
		t.Fatalf("expected ErrBioTooLong, got %v", err)
	}
}

func TestNormalizePreferences(t *testing.T) {
	ok, err := normalizePreferences(Preferences{InterestedIn: []string{"woman", "woman", "man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(ok.InterestedIn) != 2 {
		t.Fatalf("got %+v %v", ok, err)
	}
	bad := []Preferences{
		{InterestedIn: []string{"woman"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"woman"}, MinAge: 30, MaxAge: 20, MaxDistanceKm: 30},
		{InterestedIn: []string{"woman"}, MinAge: 20, MaxAge: 100, MaxDistanceKm: 30},
		{InterestedIn: []string{"woman"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"robot"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: nil, MinAge: 20, MaxAge: 30, MaxDistanceKm: 10},
	}
	for i, p := range bad {
		if _, err := normalizePreferences(p); err == nil {
			t.Errorf("case %d must be rejected", i)
		}
	}
}

func TestLocationIsRoundedAndDistanceIsBucketed(t *testing.T) {
	lat, lng, city, err := normalizeLocation(48.856614, 2.352222, "  Paris ")
	if err != nil || lat != 48.86 || lng != 2.35 || city != "Paris" {
		t.Fatalf("got %v %v %q %v", lat, lng, city, err)
	}
	if _, _, _, err := normalizeLocation(91, 0, ""); err == nil {
		t.Fatal("latitude out of range must be rejected")
	}
	for in, want := range map[float64]int{0: 5, 4.9: 5, 5.1: 10, 12: 15, 47.2: 50} {
		if got := ApproxDistanceKm(in); got != want {
			t.Errorf("ApproxDistanceKm(%v)=%d want %d", in, got, want)
		}
	}
}
