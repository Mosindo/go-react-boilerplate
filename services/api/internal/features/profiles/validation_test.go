package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := map[string]int{"2008-06-15": 18, "2008-06-16": 17, "1990-01-01": 36, "2000-12-31": 25}
	for birth, want := range cases {
		b, _ := time.Parse(birthdateLayout, birth)
		if got := AgeAt(b, now); got != want {
			t.Errorf("AgeAt(%s) = %d, want %d", birth, got, want)
		}
	}
}

func TestValidateProfileRejectsMinorsAndBadInput(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	base := UpdateProfileRequest{FirstName: "Camille", Birthdate: "1995-03-02", Gender: "woman", Interests: []string{"hiking", "hiking", "music"}}

	got, err := ValidateProfile(base, now)
	if err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	if len(got.Interests) != 2 || got.LookingFor != "not_sure" {
		t.Fatalf("unexpected normalisation: %+v", got)
	}

	minor := base
	minor.Birthdate = "2008-06-16"
	var verr *ValidationError
	if _, err := ValidateProfile(minor, now); !errors.As(err, &verr) || verr.Code != "underage" {
		t.Fatalf("expected underage error, got %v", err)
	}

	for name, mutate := range map[string]func(*UpdateProfileRequest){
		"empty name":   func(r *UpdateProfileRequest) { r.FirstName = "  " },
		"digits name":  func(r *UpdateProfileRequest) { r.FirstName = "R2D2" },
		"bad gender":   func(r *UpdateProfileRequest) { r.Gender = "robot" },
		"bad date":     func(r *UpdateProfileRequest) { r.Birthdate = "02/03/1995" },
		"too old":      func(r *UpdateProfileRequest) { r.Birthdate = "1900-01-01" },
		"bad height":   func(r *UpdateProfileRequest) { h := 20; r.HeightCm = &h },
		"long bio":     func(r *UpdateProfileRequest) { r.Bio = strings.Repeat("é", 501) },
		"bad relation": func(r *UpdateProfileRequest) { r.LookingFor = "anything" },
		"too many interests": func(r *UpdateProfileRequest) {
			r.Interests = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
		},
	} {
		req := base
		mutate(&req)
		if _, err := ValidateProfile(req, now); err == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	ok, err := ValidatePreferences(UpdatePreferencesRequest{InterestedIn: []string{"man", "woman", "man"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(ok.InterestedIn) != 2 {
		t.Fatalf("unexpected result %+v %v", ok, err)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 501},
	}
	for i, req := range bad {
		if _, err := ValidatePreferences(req); err == nil {
			t.Errorf("case %d: expected error", i)
		}
	}
}

func TestCoarsenLocationAndDistance(t *testing.T) {
	lat, lng, err := CoarsenLocation(48.856613, 2.352222)
	if err != nil || lat != 48.86 || lng != 2.35 {
		t.Fatalf("unexpected coarsening %v %v %v", lat, lng, err)
	}
	if _, _, err := CoarsenLocation(91, 0); err == nil {
		t.Fatal("expected out of range latitude to fail")
	}
	for km, want := range map[float64]int{0.2: 1, 3.4: 3, 19.6: 20, 22: 20, 23: 25, 151: 150} {
		if got := ApproximateDistance(km); got != want {
			t.Errorf("ApproximateDistance(%v) = %d, want %d", km, got, want)
		}
	}
}
