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
	cases := map[string]int{
		"2008-06-15": 18, // birthday today
		"2008-06-16": 17, // birthday tomorrow
		"2000-01-01": 26,
		"2000-12-31": 25,
	}
	for birth, want := range cases {
		if got := AgeOn(date(birth), now); got != want {
			t.Errorf("AgeOn(%s) = %d, want %d", birth, got, want)
		}
	}
}

func TestValidateProfileUpdate(t *testing.T) {
	now := date("2026-06-15")
	h := 175
	good, err := ValidateProfileUpdate(UpdateProfileRequest{
		FirstName: "  Anne-Marie ", Birthdate: "1990-05-01", Gender: "woman",
		Bio: "Hello\n\n\n\nworld", HeightCm: &h, InterestIDs: []int{3, 3, 5},
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if good.FirstName != "Anne-Marie" || good.Bio != "Hello\n\nworld" || len(good.InterestIDs) != 2 {
		t.Fatalf("unexpected normalisation: %+v", good)
	}

	short := 90
	bad := []struct {
		name string
		req  UpdateProfileRequest
		want error
	}{
		{"underage", UpdateProfileRequest{Birthdate: "2010-01-01"}, ErrUnderage},
		{"bad date", UpdateProfileRequest{Birthdate: "1990-02-30"}, ErrInvalidBirthdate},
		{"too old", UpdateProfileRequest{Birthdate: "1900-01-01"}, ErrInvalidBirthdate},
		{"gender", UpdateProfileRequest{Gender: "other"}, ErrInvalidGender},
		{"name digits", UpdateProfileRequest{FirstName: "B0b"}, ErrInvalidName},
		{"name html", UpdateProfileRequest{FirstName: "<b>"}, ErrInvalidName},
		{"height", UpdateProfileRequest{HeightCm: &short}, ErrInvalidHeight},
		{"interests", UpdateProfileRequest{InterestIDs: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}}, ErrTooManyInterests},
		{"interest id", UpdateProfileRequest{InterestIDs: []int{0}}, ErrUnknownInterest},
	}
	for _, c := range bad {
		if _, err := ValidateProfileUpdate(c.req, now); err != c.want {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

func TestEmptyInterestsAreNonNil(t *testing.T) {
	u, err := ValidateProfileUpdate(UpdateProfileRequest{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if u.InterestIDs == nil {
		t.Fatal("interest ids must be an empty slice so the repository clears them")
	}
}

func TestValidatePreferences(t *testing.T) {
	ok, err := ValidatePreferences(UpdatePreferencesRequest{InterestedIn: []string{"man", "man", "woman"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(ok.InterestedIn) != 2 {
		t.Fatalf("unexpected: %+v %v", ok, err)
	}
	for _, req := range []UpdatePreferencesRequest{
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 30, MaxAge: 20, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 100, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"alien"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 10},
	} {
		if _, err := ValidatePreferences(req); err == nil {
			t.Errorf("expected rejection for %+v", req)
		}
	}
}

func TestApproximateCoordinatesAndDistance(t *testing.T) {
	lat, lng, err := ApproximateCoordinates(48.858370, 2.294481)
	if err != nil || lat != 48.86 || lng != 2.29 {
		t.Fatalf("unexpected rounding: %v %v %v", lat, lng, err)
	}
	if _, _, err := ApproximateCoordinates(91, 0); err == nil {
		t.Fatal("expected invalid latitude")
	}
	if DisplayDistance(0.2) != 1 || DisplayDistance(3.1) != 4 {
		t.Fatal("distance must be rounded up with a 1 km floor")
	}
}

func TestComputeCompleteness(t *testing.T) {
	c := ComputeCompleteness(OwnProfile{})
	if c.IsComplete || len(c.Missing) != 4 {
		t.Fatalf("unexpected: %+v", c)
	}
	g := "man"
	b := date("1990-01-01")
	c = ComputeCompleteness(OwnProfile{FirstName: "Léo", Gender: &g, Birthdate: &b, Photos: []Photo{{ID: "p"}}})
	if !c.IsComplete {
		t.Fatalf("expected complete: %+v", c)
	}
}
