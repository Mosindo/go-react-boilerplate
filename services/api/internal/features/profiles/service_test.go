package profiles

import (
	"errors"
	"math"
	"testing"
	"time"
)

var refNow = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func validInput() UpsertProfileInput {
	return UpsertProfileInput{FirstName: "  Camille  ", Birthdate: "1995-03-02", Gender: "woman", Bio: " Hello ", InterestIDs: []int{3, 3, 5}}
}

func TestValidateProfileInputNormalizes(t *testing.T) {
	out, birthdate, err := ValidateProfileInput(validInput(), refNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.FirstName != "Camille" || out.Bio != "Hello" {
		t.Fatalf("expected trimmed fields, got %#v", out)
	}
	if len(out.InterestIDs) != 2 {
		t.Fatalf("expected deduplicated interests, got %v", out.InterestIDs)
	}
	if birthdate.Year() != 1995 {
		t.Fatalf("unexpected birthdate %v", birthdate)
	}
}

func TestValidateProfileInputRejectsUnderage(t *testing.T) {
	in := validInput()
	in.Birthdate = "2008-06-16" // turns 18 the day after refNow
	if _, _, err := ValidateProfileInput(in, refNow); !errors.Is(err, ErrUnderage) {
		t.Fatalf("expected ErrUnderage, got %v", err)
	}
	in.Birthdate = "2008-06-15" // 18th birthday
	if _, _, err := ValidateProfileInput(in, refNow); err != nil {
		t.Fatalf("18th birthday must be accepted: %v", err)
	}
}

func TestValidateProfileInputRejectsInvalidValues(t *testing.T) {
	cases := map[string]func(*UpsertProfileInput){
		"empty name":    func(in *UpsertProfileInput) { in.FirstName = "   " },
		"bad date":      func(in *UpsertProfileInput) { in.Birthdate = "1995-02-30" },
		"bad gender":    func(in *UpsertProfileInput) { in.Gender = "robot" },
		"bad height":    func(in *UpsertProfileInput) { h := 300; in.HeightCm = &h },
		"bad goal":      func(in *UpsertProfileInput) { g := "forever"; in.RelationshipGoal = &g },
		"too many tags": func(in *UpsertProfileInput) { in.InterestIDs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} },
		"negative tag":  func(in *UpsertProfileInput) { in.InterestIDs = []int{-1} },
	}
	for name, mutate := range cases {
		in := validInput()
		mutate(&in)
		if _, _, err := ValidateProfileInput(in, refNow); err == nil {
			t.Fatalf("%s: expected validation error", name)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	out, err := ValidatePreferences(Preferences{InterestedIn: []string{"man", "man", "woman"}, AgeMin: 25, AgeMax: 40, MaxDistanceKm: 30})
	if err != nil || len(out.InterestedIn) != 2 {
		t.Fatalf("expected valid deduplicated prefs, got %#v %v", out, err)
	}
	bad := []Preferences{
		{InterestedIn: nil, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 17, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 40, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 501},
	}
	for i, prefs := range bad {
		if _, err := ValidatePreferences(prefs); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestAgeOn(t *testing.T) {
	birth := time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC)
	if AgeOn(birth, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)) != 25 {
		t.Fatal("birthday not reached yet")
	}
	if AgeOn(birth, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) != 26 {
		t.Fatal("birthday passed")
	}
}

func TestGeoHelpers(t *testing.T) {
	if got := SnapCoordinate(48.85661); math.Abs(got-48.86) > 1e-9 {
		t.Fatalf("unexpected snap %v", got)
	}
	paris, lyon := [2]float64{48.8566, 2.3522}, [2]float64{45.764, 4.8357}
	km := HaversineKm(paris[0], paris[1], lyon[0], lyon[1])
	if km < 385 || km > 400 {
		t.Fatalf("unexpected Paris-Lyon distance %v", km)
	}
	if ApproximateKm(0.2) != 1 || ApproximateKm(4.4) != 4 || ApproximateKm(23) != 25 {
		t.Fatal("unexpected approximate distances")
	}
	if ValidCoordinates(91, 0) || ValidCoordinates(0, 181) || !ValidCoordinates(-33.9, 151.2) {
		t.Fatal("unexpected coordinate validation")
	}
}
