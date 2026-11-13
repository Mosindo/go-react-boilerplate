package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func ptr[T any](v T) *T { return &v }

func TestAgeOn(t *testing.T) {
	birth := time.Date(2000, 6, 16, 0, 0, 0, 0, time.UTC)
	if got := AgeOn(birth, now); got != 25 {
		t.Fatalf("day before birthday: got %d", got)
	}
	if got := AgeOn(birth, now.AddDate(0, 0, 1)); got != 26 {
		t.Fatalf("on birthday: got %d", got)
	}
}

func TestValidatePatchAgeGate(t *testing.T) {
	cases := map[string]error{
		"2008-06-16": ErrTooYoung, // 17 the day before the 18th birthday
		"2008-06-15": nil,         // exactly 18 today
		"1900-01-01": ErrInvalidAge,
		"2030-01-01": ErrInvalidBirthdate,
		"15/06/1990": ErrInvalidBirthdate,
	}
	for in, want := range cases {
		_, err := validatePatch(UpdateProfileRequest{Birthdate: ptr(in)}, now)
		if !errors.Is(err, want) && !(want == nil && err == nil) {
			t.Errorf("birthdate %s: want %v, got %v", in, want, err)
		}
	}
}

func TestValidatePatchFields(t *testing.T) {
	p, err := validatePatch(UpdateProfileRequest{
		FirstName:        ptr("  Marie-Hélène  "),
		Gender:           ptr("WOMAN"),
		Bio:              ptr("  Hello  "),
		HeightCm:         ptr(0),
		RelationshipGoal: ptr(""),
		Interests:        ptr([]string{"Hiking", "hiking", "jazz"}),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if *p.FirstName != "Marie-Hélène" || *p.Gender != "woman" || *p.Bio != "Hello" {
		t.Fatalf("normalization failed: %+v", p)
	}
	if !p.ClearHeight || !p.ClearRelationGoal {
		t.Fatal("zero height and empty goal clear the fields")
	}
	if len(*p.InterestSlugs) != 2 {
		t.Fatalf("interests must be deduplicated, got %v", *p.InterestSlugs)
	}

	bad := []UpdateProfileRequest{
		{FirstName: ptr("")},
		{FirstName: ptr("Robert'); DROP TABLE users;--")},
		{FirstName: ptr(strings.Repeat("a", 41))},
		{Gender: ptr("other")},
		{Bio: ptr(strings.Repeat("é", 501))},
		{HeightCm: ptr(300)},
		{RelationshipGoal: ptr("marriage-now")},
		{Interests: ptr([]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"})},
	}
	for i, req := range bad {
		if _, err := validatePatch(req, now); err == nil || !IsValidationError(err) {
			t.Errorf("case %d should fail with a validation error, got %v", i, err)
		}
	}
}

func TestValidateLocationRoundsCoordinates(t *testing.T) {
	lat, lon, city, err := validateLocation(48.856613, 2.352222, "  Paris  ")
	if err != nil || lat != 48.86 || lon != 2.35 || city != "Paris" {
		t.Fatalf("got %v %v %q %v", lat, lon, city, err)
	}
	if _, _, _, err := validateLocation(91, 0, ""); !errors.Is(err, ErrInvalidLocation) {
		t.Fatal("latitude 91 must be rejected")
	}
}

func TestApplyPreferences(t *testing.T) {
	base := Preferences{InterestedIn: []string{}, AgeMin: 18, AgeMax: 99, MaxDistanceKm: 50}
	next, err := applyPreferences(base, UpdatePreferencesRequest{InterestedIn: ptr([]string{"man", "woman", "man"}), AgeMin: ptr(25), AgeMax: ptr(35)})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.InterestedIn) != 2 || next.AgeMin != 25 || next.AgeMax != 35 {
		t.Fatalf("unexpected %+v", next)
	}
	for _, req := range []UpdatePreferencesRequest{
		{InterestedIn: ptr([]string{})},
		{AgeMin: ptr(17)},
		{AgeMin: ptr(40), AgeMax: ptr(30)},
		{MaxDistanceKm: ptr(0)},
		{MaxDistanceKm: ptr(1000)},
	} {
		if _, err := applyPreferences(base, req); err == nil {
			t.Errorf("expected error for %+v", req)
		}
	}
}

func TestApproximateDistance(t *testing.T) {
	cases := map[float64]int{0.2: 1, 1.0: 1, 1.1: 2, 9.5: 10, 10.2: 15, 23: 25, 49.9: 50}
	for in, want := range cases {
		if got := ApproximateDistance(in); got != want {
			t.Errorf("ApproximateDistance(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestMissingFields(t *testing.T) {
	missing := missingFields(storedProfile{}, 0)
	if len(missing) != 6 {
		t.Fatalf("empty profile should miss 6 fields, got %v", missing)
	}
	birth := now
	gender := "man"
	complete := storedProfile{FirstName: "Léo", Birthdate: &birth, Gender: &gender, HasLocation: true,
		Preferences: Preferences{InterestedIn: []string{"woman"}}}
	if m := missingFields(complete, 1); len(m) != 0 {
		t.Fatalf("complete profile should miss nothing, got %v", m)
	}
}
