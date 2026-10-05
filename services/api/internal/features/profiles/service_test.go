package profiles

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	at := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	for birth, want := range map[string]int{
		"2008-06-15": 18, // birthday today
		"2008-06-16": 17, // birthday tomorrow
		"2008-06-14": 18,
		"1990-12-31": 35,
		"2000-02-29": 26,
	} {
		b, _ := time.Parse("2006-01-02", birth)
		if got := AgeOn(b, at); got != want {
			t.Errorf("AgeOn(%s) = %d, want %d", birth, got, want)
		}
	}
}

func TestRoundDistanceKm(t *testing.T) {
	for km, want := range map[float64]int{0: 5, 0.3: 5, 5: 5, 5.1: 10, 10: 10, 12.4: 15, 99.9: 100} {
		if got := RoundDistanceKm(km); got != want {
			t.Errorf("RoundDistanceKm(%v) = %d, want %d", km, got, want)
		}
	}
}

func TestValidateProfile(t *testing.T) {
	s := NewService(nil)
	s.now = func() time.Time { return time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) }
	ok := UpsertProfileRequest{FirstName: " Jean-Luc ", BirthDate: "1990-01-01", Gender: "man", Bio: "  bonjour\x00 ", Occupation: "  Dev   web ", Interests: []string{"travel", "travel", " music "}}

	in, err := s.validateProfile(ok)
	if err != nil {
		t.Fatal(err)
	}
	if in.FirstName != "Jean-Luc" || in.Bio != "bonjour" || in.Occupation != "Dev web" || len(in.Interests) != 2 {
		t.Fatalf("input must be normalised: %+v", in)
	}

	bad := map[string]func(*UpsertProfileRequest){
		"18th birthday tomorrow": func(r *UpsertProfileRequest) { r.BirthDate = "2008-06-16" },
		"empty name":             func(r *UpsertProfileRequest) { r.FirstName = "   " },
		"emoji name":             func(r *UpsertProfileRequest) { r.FirstName = "Zoé😀" },
		"future":                 func(r *UpsertProfileRequest) { r.BirthDate = "2030-01-01" },
		"gender":                 func(r *UpsertProfileRequest) { r.Gender = "x" },
	}
	for name, mutate := range bad {
		r := ok
		mutate(&r)
		if _, err := s.validateProfile(r); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	r := ok
	r.BirthDate = "2008-06-15"
	if _, err := s.validateProfile(r); err != nil {
		t.Errorf("exactly 18 today must be accepted: %v", err)
	}
}
