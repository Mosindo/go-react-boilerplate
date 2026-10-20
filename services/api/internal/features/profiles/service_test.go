package profiles

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	cases := map[string]int{
		"2008-03-10": 18, // birthday today
		"2008-03-11": 17, // birthday tomorrow
		"2000-12-31": 25,
		"1990-03-09": 36,
		"2024-02-29": 2, // leap day
	}
	for birth, want := range cases {
		b, _ := time.Parse("2006-01-02", birth)
		if got := AgeOn(b, now); got != want {
			t.Errorf("AgeOn(%s) = %d, want %d", birth, got, want)
		}
	}
}

func TestBucketDistanceNeverReturnsExactValues(t *testing.T) {
	cases := map[float64]int{0: 1, 0.4: 1, 1.01: 2, 2.7: 3, 4: 5, 7.3: 10, 11: 15, 20: 25, 26: 50, 99.9: 100, 100.1: 150, 320: 350}
	for km, want := range cases {
		if got := BucketDistance(km); got != want {
			t.Errorf("BucketDistance(%v) = %d, want %d", km, got, want)
		}
	}
}

func TestValidateProfileAgeBoundary(t *testing.T) {
	s := &Service{now: func() time.Time { return time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC) }}
	in := ProfileInput{FirstName: "Léa-Marie", Gender: GenderWoman, Interests: []string{"music", "music", " ", "art"}}

	in.BirthDate = "2008-03-10"
	stored, slugs, err := s.validateProfile(in)
	if err != nil {
		t.Fatalf("18th birthday today must pass: %v", err)
	}
	if len(slugs) != 2 || stored.FirstName != "Léa-Marie" {
		t.Fatalf("interests must be deduplicated: %v", slugs)
	}
	in.BirthDate = "2008-03-11"
	if _, _, err := s.validateProfile(in); err == nil {
		t.Fatal("17 year old must be rejected")
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("7b0f6d3e-1c1a-4a0e-9d57-0f9d3b7a1e55") || IsUUID("1; DROP TABLE users") || IsUUID("") {
		t.Fatal("uuid guard is wrong")
	}
}
