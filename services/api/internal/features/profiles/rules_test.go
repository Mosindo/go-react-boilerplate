package profiles

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestAgeOn(t *testing.T) {
	now := date("2026-06-15")
	cases := map[string]int{"2008-06-15": 18, "2008-06-16": 17, "1990-01-01": 36, "2000-02-29": 26}
	for b, want := range cases {
		if got := AgeOn(date(b), now); got != want {
			t.Fatalf("AgeOn(%s) = %d, want %d", b, got, want)
		}
	}
}

func TestBirthdateBoundsMatchAges(t *testing.T) {
	now := date("2026-06-15")
	earliest, latest := BirthdateBounds(25, 30, now)
	if AgeOn(latest, now) != 25 || AgeOn(latest.AddDate(0, 0, 1), now) != 24 {
		t.Fatalf("latest bound wrong: %v", latest)
	}
	if AgeOn(earliest, now) != 30 || AgeOn(earliest.AddDate(0, 0, -1), now) != 31 {
		t.Fatalf("earliest bound wrong: %v", earliest)
	}
}

func TestApproxDistanceAndRounding(t *testing.T) {
	cases := map[float64]int{0.2: 1, 1.9: 1, 3.4: 3, 9.6: 10, 12: 10, 13: 15, 47.6: 50}
	for in, want := range cases {
		if got := ApproxDistanceKm(in); got != want {
			t.Fatalf("ApproxDistanceKm(%v) = %d, want %d", in, got, want)
		}
	}
	if RoundCoordinate(45.764043) != 45.76 || RoundCoordinate(-4.835659) != -4.84 {
		t.Fatal("coordinates must be rounded to 2 decimals")
	}
	if _, _, err := ValidateCoordinates(91, 0); err == nil {
		t.Fatal("latitude 91 must be rejected")
	}
}

func TestValidateProfile(t *testing.T) {
	now := date("2026-06-15")
	in, err := ValidateProfile(UpsertProfileRequest{FirstName: "  Jean   Luc ", Birthdate: "1990-01-01", Gender: "man", Bio: "a\n\n\n\nb\x07", City: " Paris "}, now)
	if err != nil {
		t.Fatal(err)
	}
	if in.FirstName != "Jean Luc" || in.Bio != "a\n\nb" || in.City != "Paris" || in.RelationshipGoal != "unsure" {
		t.Fatalf("unexpected normalization: %+v", in)
	}
	bad := []UpsertProfileRequest{
		{FirstName: "", Birthdate: "1990-01-01", Gender: "man"},
		{FirstName: "R2D2", Birthdate: "1990-01-01", Gender: "man"},
		{FirstName: "Al", Birthdate: "2010-01-01", Gender: "man"},
		{FirstName: "Al", Birthdate: "1990-13-01", Gender: "man"},
		{FirstName: "Al", Birthdate: "1990-01-01", Gender: "other"},
		{FirstName: "Al", Birthdate: "1990-01-01", Gender: "man", RelationshipGoal: "marriage"},
		{FirstName: "Al", Birthdate: "1890-01-01", Gender: "man"},
	}
	for _, req := range bad {
		if _, err := ValidateProfile(req, now); err == nil {
			t.Fatalf("expected rejection for %+v", req)
		}
	}
}

func TestValidatePreferencesAndInterests(t *testing.T) {
	p, err := ValidatePreferences(PreferencesRequest{InterestedIn: []string{"woman", "man", "woman"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 25})
	if err != nil || len(p.InterestedIn) != 2 {
		t.Fatalf("unexpected: %+v %v", p, err)
	}
	for _, req := range []PreferencesRequest{
		{InterestedIn: nil, MinAge: 20, MaxAge: 40, MaxDistanceKm: 25},
		{InterestedIn: []string{"cat"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 25},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 25},
		{InterestedIn: []string{"man"}, MinAge: 30, MaxAge: 20, MaxDistanceKm: 25},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 0},
	} {
		if _, err := ValidatePreferences(req); err == nil {
			t.Fatalf("expected rejection for %+v", req)
		}
	}
	ids, err := ValidateInterestIDs([]int{1, 2, 2, 3})
	if err != nil || len(ids) != 3 {
		t.Fatalf("dedupe failed: %v %v", ids, err)
	}
	if _, err := ValidateInterestIDs([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}); err != ErrTooManyInterests {
		t.Fatal("expected too many interests")
	}
	if _, err := ValidateInterestIDs([]int{-1}); err != ErrUnknownInterest {
		t.Fatal("expected unknown interest")
	}
}

func TestMissingSteps(t *testing.T) {
	if len(missingSteps(Preferences{}, 0)) != 2 || len(missingSteps(Preferences{InterestedIn: []string{"man"}}, 1)) != 0 {
		t.Fatal("unexpected completeness evaluation")
	}
}
