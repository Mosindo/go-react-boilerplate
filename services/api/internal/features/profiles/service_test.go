package profiles

import (
	"errors"
	"math"
	"testing"
	"time"

	"example.com/api/internal/platform/httpx"
)

func TestAgeAt(t *testing.T) {
	birth := time.Date(2000, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		at   time.Time
		want int
	}{
		{time.Date(2018, 6, 14, 12, 0, 0, 0, time.UTC), 17},
		{time.Date(2018, 6, 15, 0, 0, 0, 0, time.UTC), 18},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 25},
	}
	for _, tc := range cases {
		if got := AgeAt(birth, tc.at); got != tc.want {
			t.Fatalf("AgeAt(%v) = %d, want %d", tc.at, got, tc.want)
		}
	}
}

func TestSnapCoordinateHidesExactPosition(t *testing.T) {
	lat := 48.856613
	snapped := SnapCoordinate(lat)
	if math.Abs(snapped-48.86) > 1e-9 {
		t.Fatalf("expected 48.86, got %v", snapped)
	}
	if math.Abs(SnapCoordinate(2.352222)-2.35) > 1e-9 {
		t.Fatalf("unexpected longitude snap")
	}
}

func TestDistanceAndDisplay(t *testing.T) {
	// Paris -> Lyon is about 392 km.
	d := DistanceKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("unexpected Paris-Lyon distance %v", d)
	}
	if DisplayDistance(0.2) != 1 {
		t.Fatal("distances under 1.5 km must display as 1 km")
	}
	if DisplayDistance(7.4) != 7 {
		t.Fatal("expected 7 km")
	}
	if DisplayDistance(392.4) != 390 {
		t.Fatalf("long distances are rounded to 5 km, got %d", DisplayDistance(392.4))
	}
}

func TestValidatePreferences(t *testing.T) {
	prefs, err := ValidatePreferences(PreferencesRequest{InterestedIn: []string{"woman", "woman", "man"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prefs.InterestedIn) != 2 {
		t.Fatalf("duplicates must be removed: %v", prefs.InterestedIn)
	}

	invalid := []PreferencesRequest{
		{InterestedIn: []string{}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 501},
	}
	for i, req := range invalid {
		var v *httpx.ValidationError
		if _, err := ValidatePreferences(req); !errors.As(err, &v) {
			t.Fatalf("case %d: expected validation error, got %v", i, err)
		}
	}
}

func TestNormalizeFirstName(t *testing.T) {
	name, err := normalizeFirstName("  Anne-Marie   O'Neil ")
	if err != nil || name != "Anne-Marie O'Neil" {
		t.Fatalf("got %q, %v", name, err)
	}
	for _, bad := range []string{"", "   ", "<script>", "R2D2", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		if _, err := normalizeFirstName(bad); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

func TestParseBirthdateEnforcesAdultsOnly(t *testing.T) {
	s := &Service{now: func() time.Time { return time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC) }}
	if _, err := s.parseBirthdate("2008-06-16"); err == nil {
		t.Fatal("17-year-olds must be rejected")
	}
	if _, err := s.parseBirthdate("2008-06-15"); err != nil {
		t.Fatalf("18th birthday must be accepted: %v", err)
	}
	if _, err := s.parseBirthdate("15/06/2000"); err == nil {
		t.Fatal("invalid format must be rejected")
	}
	if _, err := s.parseBirthdate("1890-01-01"); err == nil {
		t.Fatal("implausible ages must be rejected")
	}
}

func TestCleanTextStripsControlCharacters(t *testing.T) {
	got := cleanText("Hello\x00 world\r\n\n\n\n\nbye​")
	if got != "Hello world\n\nbye​" {
		t.Fatalf("unexpected cleaned text %q", got)
	}
}
