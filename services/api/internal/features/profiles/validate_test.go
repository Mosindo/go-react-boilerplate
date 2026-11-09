package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func validInput() ProfileInput {
	return ProfileInput{FirstName: "  Léa ", BirthDate: "1995-06-15", Gender: "Woman", Bio: "Hello", City: "Lyon"}
}

func TestAgeOnBirthdayBoundary(t *testing.T) {
	if got := AgeOn(time.Date(2008, 6, 16, 0, 0, 0, 0, time.UTC), now); got != 17 {
		t.Fatalf("day before 18th birthday: got %d", got)
	}
	if got := AgeOn(time.Date(2008, 6, 15, 0, 0, 0, 0, time.UTC), now); got != 18 {
		t.Fatalf("18th birthday: got %d", got)
	}
}

func TestNormalizeProfileAcceptsAndNormalizes(t *testing.T) {
	got, _, err := NormalizeProfile(validInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	if got.FirstName != "Léa" || got.Gender != "woman" {
		t.Fatalf("not normalized: %+v", got)
	}
}

func TestNormalizeProfileRejects(t *testing.T) {
	lat := 10.0
	cases := map[string]func(*ProfileInput){
		"underage":     func(p *ProfileInput) { p.BirthDate = "2010-01-01" },
		"bad date":     func(p *ProfileInput) { p.BirthDate = "01/01/1990" },
		"too old":      func(p *ProfileInput) { p.BirthDate = "1900-01-01" },
		"empty name":   func(p *ProfileInput) { p.FirstName = "   " },
		"long name":    func(p *ProfileInput) { p.FirstName = strings.Repeat("a", 41) },
		"bad gender":   func(p *ProfileInput) { p.Gender = "robot" },
		"long bio":     func(p *ProfileInput) { p.Bio = strings.Repeat("a", 501) },
		"dup interest": func(p *ProfileInput) { p.InterestIDs = []int{1, 1} },
		"many":         func(p *ProfileInput) { p.InterestIDs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} },
		"half coords":  func(p *ProfileInput) { p.Latitude = &lat },
		"control char": func(p *ProfileInput) { p.FirstName = "a\x00b" },
	}
	for name, mutate := range cases {
		in := validInput()
		mutate(&in)
		if _, _, err := NormalizeProfile(in, now); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: expected validation error, got %v", name, err)
		}
	}
}

func TestNormalizeProfileRoundsCoordinates(t *testing.T) {
	in := validInput()
	lat, lon := 45.764043, 4.835659
	in.Latitude, in.Longitude = &lat, &lon
	got, _, err := NormalizeProfile(in, now)
	if err != nil {
		t.Fatal(err)
	}
	if *got.Latitude != 45.76 || *got.Longitude != 4.84 {
		t.Fatalf("coords not rounded: %v %v", *got.Latitude, *got.Longitude)
	}
}

func TestNormalizePreferences(t *testing.T) {
	ok, err := NormalizePreferences(Preferences{InterestedIn: []string{"Man", "man", "woman"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(ok.InterestedIn) != 2 {
		t.Fatalf("unexpected: %+v %v", ok, err)
	}
	bad := []Preferences{
		{InterestedIn: []string{"x"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 20, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 501},
	}
	for i, p := range bad {
		if _, err := NormalizePreferences(p); !errors.Is(err, ErrValidation) {
			t.Errorf("case %d accepted", i)
		}
	}
}
