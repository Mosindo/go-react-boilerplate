package profiles

import (
	"math"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func validInput() ProfileInput {
	return ProfileInput{FirstName: "  Camille ", Birthdate: "1995-03-02", Gender: "woman", Bio: " Hello\n", InterestIDs: []int{1, 2, 2}}
}

func TestAgeOn(t *testing.T) {
	b := time.Date(2008, 6, 16, 0, 0, 0, 0, time.UTC)
	if AgeOn(b, now) != 17 {
		t.Fatal("birthday tomorrow: still 17")
	}
	b = time.Date(2008, 6, 15, 0, 0, 0, 0, time.UTC)
	if AgeOn(b, now) != 18 {
		t.Fatal("birthday today: 18")
	}
}

func TestValidateProfile(t *testing.T) {
	rec, err := validateProfile(validInput(), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.FirstName != "Camille" || rec.Bio != "Hello" || rec.LookingFor != "unsure" || len(rec.InterestIDs) != 2 {
		t.Fatalf("unexpected normalization: %#v", rec)
	}

	cases := map[string]func(*ProfileInput){
		"minor":        func(p *ProfileInput) { p.Birthdate = "2008-06-16" },
		"bad date":     func(p *ProfileInput) { p.Birthdate = "02/03/1995" },
		"too old":      func(p *ProfileInput) { p.Birthdate = "1900-01-01" },
		"empty name":   func(p *ProfileInput) { p.FirstName = "  " },
		"digit name":   func(p *ProfileInput) { p.FirstName = "R2D2" },
		"bad gender":   func(p *ProfileInput) { p.Gender = "robot" },
		"bad intent":   func(p *ProfileInput) { p.LookingFor = "money" },
		"bad height":   func(p *ProfileInput) { h := 300; p.HeightCm = &h },
		"bad interest": func(p *ProfileInput) { p.InterestIDs = []int{-1} },
		"too many":     func(p *ProfileInput) { p.InterestIDs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} },
		"bio too long": func(p *ProfileInput) { p.Bio = strings.Repeat("é", 501) },
	}
	for name, mutate := range cases {
		in := validInput()
		mutate(&in)
		if _, err := validateProfile(in, now); !IsValidationError(err) {
			t.Errorf("%s: expected validation error, got %v", name, err)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	p, err := validatePreferences(PreferencesInput{InterestedIn: []string{"man", "woman", "man"}, AgeMin: 25, AgeMax: 40, MaxDistanceKm: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.InterestedIn) != 2 || p.InterestedIn[0] != "man" {
		t.Fatalf("unexpected %v", p.InterestedIn)
	}
	bad := []PreferencesInput{
		{InterestedIn: nil, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 17, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 40, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 501},
	}
	for i, in := range bad {
		if _, err := validatePreferences(in); !IsValidationError(err) {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}

func TestLocationIsCoarsened(t *testing.T) {
	lat, lng, err := validateLocation(48.856613, 2.352222)
	if err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("expected coarsened coordinates, got %v %v", lat, lng)
	}
	if _, _, err := validateLocation(91, 0); err == nil {
		t.Fatal("expected invalid latitude")
	}
	if _, _, err := validateLocation(math.NaN(), 0); err == nil {
		t.Fatal("expected NaN to be rejected")
	}
}

func TestApproximateDistance(t *testing.T) {
	cases := map[float64]int{0.2: 1, 1.4: 1, 3.6: 4, 12: 10, 13: 15, 87.4: 85}
	for in, want := range cases {
		if got := ApproximateDistanceKm(in); got != want {
			t.Errorf("ApproximateDistanceKm(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestHaversine(t *testing.T) {
	// Paris -> Lyon is ~392 km.
	d := HaversineKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("unexpected distance %v", d)
	}
}
