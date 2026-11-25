package profiles

import (
	"testing"
	"time"
)

func TestAgeAt(t *testing.T) {
	birth := time.Date(2000, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := map[time.Time]int{
		time.Date(2018, 6, 14, 0, 0, 0, 0, time.UTC): 17,
		time.Date(2018, 6, 15, 0, 0, 0, 0, time.UTC): 18,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC):  25,
	}
	for now, want := range cases {
		if got := AgeAt(birth, now); got != want {
			t.Fatalf("AgeAt(%v) = %d, want %d", now, got, want)
		}
	}
}

func TestCoarsenLocation(t *testing.T) {
	lat, lon, err := CoarsenLocation(48.856614, 2.3522219)
	if err != nil || lat != 48.86 || lon != 2.35 {
		t.Fatalf("unexpected %v %v %v", lat, lon, err)
	}
	if _, _, err := CoarsenLocation(91, 0); err == nil {
		t.Fatal("expected invalid latitude")
	}
	if _, _, err := CoarsenLocation(0, -181); err == nil {
		t.Fatal("expected invalid longitude")
	}
}

func TestApproximateDistance(t *testing.T) {
	cases := map[float64]int{0.2: 1, 1.9: 1, 2.4: 2, 12.6: 13, 21: 20, 23: 25, 147: 145}
	for in, want := range cases {
		if got := ApproximateDistance(in); got != want {
			t.Fatalf("ApproximateDistance(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	ok, err := ValidatePreferences(UpdatePreferencesRequest{InterestedIn: []string{"woman", "woman", "man"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(ok.InterestedIn) != 2 {
		t.Fatalf("unexpected %v %v", ok, err)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, MinAge: 18, MaxAge: 99, MaxDistanceKm: 10},
		{InterestedIn: []string{"robot"}, MinAge: 18, MaxAge: 99, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 99, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 99, MaxDistanceKm: 0},
	}
	for _, req := range bad {
		if _, err := ValidatePreferences(req); err == nil {
			t.Fatalf("expected %+v to be rejected", req)
		}
	}
}

func TestValidateProfile(t *testing.T) {
	s := &Service{now: func() time.Time { return time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC) }}
	valid := UpdateProfileRequest{FirstName: "  Anne-Sophie ", Birthdate: "1995-04-02", Gender: "woman", Bio: "Hello\x00 there", InterestIDs: []int{1, 2}}
	in, err := s.validateProfile(valid)
	if err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	if in.FirstName != "Anne-Sophie" || in.Bio != "Hello there" {
		t.Fatalf("unexpected normalisation: %+v", in)
	}

	minor := valid
	minor.Birthdate = "2009-01-01"
	if _, err := s.validateProfile(minor); err != ErrUnderage {
		t.Fatalf("expected ErrUnderage, got %v", err)
	}

	badName := valid
	badName.FirstName = "<script>"
	if _, err := s.validateProfile(badName); err != ErrInvalidFirstName {
		t.Fatalf("expected ErrInvalidFirstName, got %v", err)
	}

	dupInterests := valid
	dupInterests.InterestIDs = []int{3, 3}
	if _, err := s.validateProfile(dupInterests); err != ErrInvalidInterests {
		t.Fatalf("expected ErrInvalidInterests, got %v", err)
	}

	badGender := valid
	badGender.Gender = "other"
	if _, err := s.validateProfile(badGender); err != ErrInvalidGender {
		t.Fatalf("expected ErrInvalidGender, got %v", err)
	}
}

func TestCompleteness(t *testing.T) {
	c := computeCompleteness(nil, Preferences{}, 0)
	if c.IsComplete || len(c.Missing) != 4 {
		t.Fatalf("unexpected %+v", c)
	}
	c = computeCompleteness(&OwnProfile{HasLocation: true}, Preferences{InterestedIn: []string{"man"}}, 1)
	if !c.IsComplete {
		t.Fatalf("expected complete, got %+v", c)
	}
}
