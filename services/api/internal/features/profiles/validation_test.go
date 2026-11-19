package profiles

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func TestAgeAt(t *testing.T) {
	cases := map[string]int{"2008-06-15": 18, "2008-06-16": 17, "1990-01-01": 36, "1990-12-31": 35}
	for raw, want := range cases {
		d, _ := time.Parse("2006-01-02", raw)
		if got := AgeAt(d, now); got != want {
			t.Fatalf("AgeAt(%s) = %d, want %d", raw, got, want)
		}
	}
}

func TestParseBirthdateEnforcesAdults(t *testing.T) {
	if _, err := ParseBirthdate("2008-06-16", now); err != ErrUnderage {
		t.Fatalf("expected ErrUnderage, got %v", err)
	}
	if _, err := ParseBirthdate("2008-06-15", now); err != nil {
		t.Fatalf("18th birthday should pass: %v", err)
	}
	if _, err := ParseBirthdate("1900-01-01", now); err != ErrInvalidAge {
		t.Fatalf("expected ErrInvalidAge, got %v", err)
	}
	if _, err := ParseBirthdate("15/06/1990", now); err != ErrInvalidBirthdate {
		t.Fatalf("expected ErrInvalidBirthdate, got %v", err)
	}
}

func validProfile() UpsertProfileRequest {
	return UpsertProfileRequest{FirstName: "  Camille  ", Birthdate: "1994-03-02", Gender: "Woman", Bio: "Hello\n\n\n\nworld", InterestIDs: []int{3, 1, 3}}
}

func TestNormalizeProfile(t *testing.T) {
	p, err := normalizeProfile(validProfile(), now)
	if err != nil {
		t.Fatal(err)
	}
	if p.FirstName != "Camille" || p.Gender != "woman" || p.LookingFor != "unsure" || p.Bio != "Hello\n\nworld" {
		t.Fatalf("unexpected normalisation: %#v", p)
	}
	if len(p.InterestIDs) != 2 || p.InterestIDs[0] != 1 || p.InterestIDs[1] != 3 {
		t.Fatalf("interests should be deduplicated and sorted: %v", p.InterestIDs)
	}
}

func TestNormalizeProfileRejectsInvalid(t *testing.T) {
	mut := []func(*UpsertProfileRequest){
		func(r *UpsertProfileRequest) { r.FirstName = "   " },
		func(r *UpsertProfileRequest) { r.FirstName = "1234" },
		func(r *UpsertProfileRequest) { r.FirstName = strings.Repeat("a", 41) },
		func(r *UpsertProfileRequest) { r.Gender = "robot" },
		func(r *UpsertProfileRequest) { r.Bio = strings.Repeat("é", 501) },
		func(r *UpsertProfileRequest) { r.LookingFor = "everything" },
		func(r *UpsertProfileRequest) { r.InterestIDs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} },
		func(r *UpsertProfileRequest) { r.InterestIDs = []int{-1} },
		func(r *UpsertProfileRequest) { r.Birthdate = "2010-01-01" },
	}
	for i, m := range mut {
		req := validProfile()
		m(&req)
		if _, err := normalizeProfile(req, now); err == nil {
			t.Fatalf("case %d should fail", i)
		}
	}
}

func TestNormalizePreferences(t *testing.T) {
	yes, no := true, false
	p, err := normalizePreferences(UpdatePreferencesRequest{InterestedIn: []string{"man", "woman", "man"}, AgeMin: 25, AgeMax: 40, MaxDistanceKm: 30, IsDiscoverable: &yes, ShowDistance: &no})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.InterestedIn) != 2 || p.ShowDistance {
		t.Fatalf("unexpected prefs %#v", p)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: []string{}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10, IsDiscoverable: &yes, ShowDistance: &yes},
		{InterestedIn: []string{"alien"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10, IsDiscoverable: &yes, ShowDistance: &yes},
		{InterestedIn: []string{"man"}, AgeMin: 17, AgeMax: 30, MaxDistanceKm: 10, IsDiscoverable: &yes, ShowDistance: &yes},
		{InterestedIn: []string{"man"}, AgeMin: 40, AgeMax: 30, MaxDistanceKm: 10, IsDiscoverable: &yes, ShowDistance: &yes},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 501, IsDiscoverable: &yes, ShowDistance: &yes},
	}
	for i, req := range bad {
		if _, err := normalizePreferences(req); err == nil {
			t.Fatalf("case %d should fail", i)
		}
	}
}
