package profiles

import (
	"errors"
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	birth := time.Date(2000, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		on   time.Time
		want int
	}{
		{time.Date(2018, 6, 14, 0, 0, 0, 0, time.UTC), 17},
		{time.Date(2018, 6, 15, 0, 0, 0, 0, time.UTC), 18},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 25},
	}
	for _, tc := range cases {
		if got := AgeOn(birth, tc.on); got != tc.want {
			t.Errorf("AgeOn(%v) = %d, want %d", tc.on, got, tc.want)
		}
	}
}

func TestParseBirthdateEnforcesAdults(t *testing.T) {
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	if _, err := ParseBirthdate("2008-11-18", now); !errors.Is(err, ErrUnderage) {
		t.Fatalf("expected underage error, got %v", err)
	}
	if _, err := ParseBirthdate("2008-11-17", now); err != nil {
		t.Fatalf("18th birthday should be accepted, got %v", err)
	}
	for _, bad := range []string{"", "17/11/1990", "1990-13-01", "1850-01-01"} {
		if _, err := ParseBirthdate(bad, now); !errors.Is(err, ErrInvalidBirthdate) {
			t.Errorf("expected invalid birthdate for %q, got %v", bad, err)
		}
	}
}

func TestNormalizeFirstName(t *testing.T) {
	if got, err := NormalizeFirstName("  Marie-  Anne "); err != nil || got != "Marie- Anne" {
		t.Fatalf("unexpected result %q %v", got, err)
	}
	if got, err := NormalizeFirstName("Zoë"); err != nil || got != "Zoë" {
		t.Fatalf("accented names must be accepted: %q %v", got, err)
	}
	for _, bad := range []string{"", "   ", "<script>", "Bob123", "a@b"} {
		if _, err := NormalizeFirstName(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestNormalizeFreeText(t *testing.T) {
	got, err := NormalizeFreeText(" hello\x00 world \n", 20, ErrBioTooLong)
	if err != nil || got != "hello world" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	if _, err := NormalizeFreeText("this is way too long", 5, ErrBioTooLong); !errors.Is(err, ErrBioTooLong) {
		t.Fatalf("expected too long error, got %v", err)
	}
}

func TestLocationPrivacyHelpers(t *testing.T) {
	if got := RoundCoordinate(48.856613); got != 48.86 {
		t.Fatalf("expected rounding to 2 decimals, got %v", got)
	}
	paris := [2]float64{48.8566, 2.3522}
	lyon := [2]float64{45.7640, 4.8357}
	d := HaversineKm(paris[0], paris[1], lyon[0], lyon[1])
	if d < 385 || d > 400 {
		t.Fatalf("unexpected Paris-Lyon distance %v", d)
	}
	if ApproximateDistanceKm(0.2) != 1 || ApproximateDistanceKm(7.4) != 7 || ApproximateDistanceKm(392) != 390 {
		t.Fatal("unexpected approximate distance buckets")
	}
	if err := ValidateCoordinates(91, 0); err == nil {
		t.Fatal("expected invalid latitude")
	}
}

func TestValidatePreferences(t *testing.T) {
	ok := UpdatePreferencesRequest{InterestedIn: []string{"woman", "nonbinary"}, AgeMin: 25, AgeMax: 40, MaxDistanceKm: 30}
	if _, err := ValidatePreferences(ok); err != nil {
		t.Fatalf("expected valid preferences, got %v", err)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man", "man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 17, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 40, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 501},
	}
	for i, req := range bad {
		if _, err := ValidatePreferences(req); !errors.Is(err, ErrInvalidPreferences) {
			t.Errorf("case %d: expected invalid preferences", i)
		}
	}
}

func TestMissingSteps(t *testing.T) {
	gender := "woman"
	birth := time.Now()
	lat, lng := 1.0, 2.0
	complete := StoredProfile{FirstName: "Ana", Birthdate: &birth, Gender: &gender, Latitude: &lat, Longitude: &lng}
	if m := MissingSteps(complete, 1); len(m) != 0 {
		t.Fatalf("expected complete profile, missing %v", m)
	}
	if m := MissingSteps(StoredProfile{}, 0); len(m) != 5 {
		t.Fatalf("expected 5 missing steps, got %v", m)
	}
}
