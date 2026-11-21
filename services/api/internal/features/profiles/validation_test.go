package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var refNow = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func baseRequest() UpsertProfileRequest {
	return UpsertProfileRequest{FirstName: "  Camille  ", Birthdate: "1995-03-10", Gender: "Woman", Bio: " Hello\r\n\n\n\nworld\x07 "}
}

func TestAge(t *testing.T) {
	birth := time.Date(2008, 6, 16, 0, 0, 0, 0, time.UTC)
	if Age(birth, refNow) != 17 {
		t.Fatal("birthday tomorrow: still 17")
	}
	if Age(time.Date(2008, 6, 15, 0, 0, 0, 0, time.UTC), refNow) != 18 {
		t.Fatal("birthday today: 18")
	}
}

func TestValidateProfileInputNormalizes(t *testing.T) {
	input, err := ValidateProfileInput(baseRequest(), refNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.FirstName != "Camille" || input.Gender != "woman" {
		t.Fatalf("unexpected normalization: %+v", input)
	}
	if input.Bio != "Hello\n\nworld" {
		t.Fatalf("unexpected bio %q", input.Bio)
	}
}

func TestValidateProfileInputRejects(t *testing.T) {
	long := strings.Repeat("a", 501)
	height := 300
	goal := "polyamory-maybe"
	cases := []struct {
		name   string
		mutate func(*UpsertProfileRequest)
		want   error
	}{
		{"minor", func(r *UpsertProfileRequest) { r.Birthdate = "2008-06-16" }, ErrTooYoung},
		{"bad date", func(r *UpsertProfileRequest) { r.Birthdate = "16/06/2000" }, ErrInvalidBirthdate},
		{"ancient", func(r *UpsertProfileRequest) { r.Birthdate = "1890-01-01" }, ErrInvalidAge},
		{"digits name", func(r *UpsertProfileRequest) { r.FirstName = "R2D2" }, ErrInvalidFirstName},
		{"empty name", func(r *UpsertProfileRequest) { r.FirstName = "   " }, ErrInvalidFirstName},
		{"gender", func(r *UpsertProfileRequest) { r.Gender = "robot" }, ErrInvalidGender},
		{"bio", func(r *UpsertProfileRequest) { r.Bio = long }, ErrBioTooLong},
		{"height", func(r *UpsertProfileRequest) { r.HeightCm = &height }, ErrInvalidHeight},
		{"goal", func(r *UpsertProfileRequest) { r.RelationshipGoal = &goal }, ErrInvalidGoal},
	}
	for _, tc := range cases {
		req := baseRequest()
		tc.mutate(&req)
		if _, err := ValidateProfileInput(req, refNow); !errors.Is(err, tc.want) {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	prefs, err := ValidatePreferences(UpdatePreferencesRequest{InterestedIn: []string{"Man", "woman", "man"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prefs.InterestedIn) != 2 || prefs.InterestedIn[0] != "man" {
		t.Fatalf("unexpected interested_in %v", prefs.InterestedIn)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: []string{}, MinAge: 18, MaxAge: 99, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 99, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 99, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 99, MaxDistanceKm: 501},
	}
	for i, req := range bad {
		if _, err := ValidatePreferences(req); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestNormalizeInterestIDs(t *testing.T) {
	ids, err := NormalizeInterestIDs([]int{3, 1, 3})
	if err != nil || len(ids) != 2 || ids[0] != 1 {
		t.Fatalf("unexpected ids %v err=%v", ids, err)
	}
	if _, err := NormalizeInterestIDs([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}); !errors.Is(err, ErrTooManyInterests) {
		t.Fatalf("expected too many interests, got %v", err)
	}
	if _, err := NormalizeInterestIDs([]int{-1}); !errors.Is(err, ErrUnknownInterest) {
		t.Fatalf("expected unknown interest, got %v", err)
	}
}
