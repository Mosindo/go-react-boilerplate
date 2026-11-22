package profiles

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeFirstName(t *testing.T) {
	ok := map[string]string{
		"  Anne-Sophie ": "Anne-Sophie",
		"D'Arcy":         "D'Arcy",
		"Zoë   Marie":    "Zoë Marie",
		"Łukasz":         "Łukasz",
	}
	for in, want := range ok {
		got, err := NormalizeFirstName(in)
		if err != nil || got != want {
			t.Errorf("NormalizeFirstName(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "   ", "Bob123", "<script>", "--", "averyveryveryveryveryveryverylongfirstnamee"} {
		if _, err := NormalizeFirstName(bad); !errors.Is(err, ErrInvalidFirstName) {
			t.Errorf("expected %q to be rejected, got %v", bad, err)
		}
	}
}

func TestParseBirthdateEnforcesAdulthood(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	if _, err := ParseBirthdate("2008-06-15", now); err != nil {
		t.Fatalf("exactly 18 today must be accepted: %v", err)
	}
	if _, err := ParseBirthdate("2008-06-16", now); !errors.Is(err, ErrUnderage) {
		t.Fatalf("one day short of 18 must be refused, got %v", err)
	}
	for _, bad := range []string{"15/06/1990", "2030-01-01", "1900-01-01", "1990-02-30"} {
		if _, err := ParseBirthdate(bad, now); !errors.Is(err, ErrInvalidBirthdate) {
			t.Errorf("expected %q to be invalid, got %v", bad, err)
		}
	}
}

func TestAgeAt(t *testing.T) {
	b := time.Date(1990, 3, 10, 0, 0, 0, 0, time.UTC)
	if got := AgeAt(b, time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)); got != 35 {
		t.Fatalf("expected 35, got %d", got)
	}
	if got := AgeAt(b, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)); got != 36 {
		t.Fatalf("expected 36, got %d", got)
	}
}

func TestRoundCoordinatesAndDistance(t *testing.T) {
	lat, lng, err := RoundCoordinates(48.856613, 2.352222)
	if err != nil || lat != 48.86 || lng != 2.35 {
		t.Fatalf("unexpected rounding: %v %v %v", lat, lng, err)
	}
	if _, _, err := RoundCoordinates(91, 0); !errors.Is(err, ErrInvalidLocation) {
		t.Fatalf("latitude > 90 must be rejected")
	}
	// Paris -> Lyon is ~392 km.
	d := DistanceKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("unexpected Paris-Lyon distance %.1f", d)
	}
	if DisplayDistanceKm(0.2) != 1 || DisplayDistanceKm(3.4) != 3 || DisplayDistanceKm(23) != 25 {
		t.Fatalf("unexpected display distance rounding")
	}
}

func TestValidatePreferences(t *testing.T) {
	p, err := ValidatePreferences(PreferencesRequest{InterestedIn: []string{"woman", "man", "woman"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil {
		t.Fatalf("valid preferences rejected: %v", err)
	}
	if len(p.InterestedIn) != 2 {
		t.Fatalf("expected de-duplicated genders, got %v", p.InterestedIn)
	}
	bad := []PreferencesRequest{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 501},
	}
	for i, req := range bad {
		if _, err := ValidatePreferences(req); !errors.Is(err, ErrInvalidPreferences) {
			t.Errorf("case %d: expected ErrInvalidPreferences, got %v", i, err)
		}
	}
}

func TestNormalizeInterestSlugs(t *testing.T) {
	got, err := NormalizeInterestSlugs([]string{" Hiking", "hiking", "music"})
	if err != nil || len(got) != 2 {
		t.Fatalf("unexpected %v %v", got, err)
	}
	many := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	if _, err := NormalizeInterestSlugs(many); !errors.Is(err, ErrTooManyInterests) {
		t.Fatalf("expected ErrTooManyInterests, got %v", err)
	}
}

func TestMissingFields(t *testing.T) {
	name, gender := "Ana", "woman"
	b := time.Now()
	lat, lng := 1.0, 2.0
	p := storedProfile{FirstName: &name, Birthdate: &b, Gender: &gender, Latitude: &lat, Longitude: &lng, PhotoCount: 1,
		Preferences: Preferences{InterestedIn: []string{"man"}}}
	if m := missingFields(p); len(m) != 0 {
		t.Fatalf("expected complete profile, missing %v", m)
	}
	p.PhotoCount = 0
	p.Latitude = nil
	if m := missingFields(p); len(m) != 2 {
		t.Fatalf("expected photos and location missing, got %v", m)
	}
}
