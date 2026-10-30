package profiles

import (
	"strings"
	"testing"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
)

var fixedNow = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func baseRequest() SaveProfileRequest {
	lat, lon := 48.8566, 2.3522
	return SaveProfileRequest{
		DisplayName: "  Camille ",
		BirthDate:   "1995-06-15",
		Gender:      "woman",
		Bio:         "Hello\nworld",
		City:        "Paris",
		Latitude:    &lat,
		Longitude:   &lon,
		Interests:   []string{"Voyage", "voyage", "cuisine"},
	}
}

func TestAgeOnHandlesBirthdayBoundary(t *testing.T) {
	birth := time.Date(2008, 6, 16, 0, 0, 0, 0, time.UTC)
	if got := AgeOn(birth, fixedNow); got != 17 {
		t.Fatalf("day before 18th birthday: want 17, got %d", got)
	}
	if got := AgeOn(birth, fixedNow.AddDate(0, 0, 1)); got != 18 {
		t.Fatalf("on 18th birthday: want 18, got %d", got)
	}
}

func TestValidateProfileNormalizesInput(t *testing.T) {
	in, err := validateProfile(baseRequest(), fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.DisplayName != "Camille" {
		t.Fatalf("name not trimmed: %q", in.DisplayName)
	}
	if len(in.InterestSlugs) != 2 || in.InterestSlugs[0] != "voyage" {
		t.Fatalf("slugs not normalized/deduplicated: %v", in.InterestSlugs)
	}
	if *in.Latitude != 48.86 || *in.Longitude != 2.35 {
		t.Fatalf("coordinates must be rounded to 2 decimals, got %v %v", *in.Latitude, *in.Longitude)
	}
}

func TestValidateProfileRejectsMinorsAndBadInput(t *testing.T) {
	cases := map[string]func(*SaveProfileRequest){
		"minor":        func(r *SaveProfileRequest) { r.BirthDate = "2008-06-16" },
		"too old":      func(r *SaveProfileRequest) { r.BirthDate = "1900-01-01" },
		"bad date":     func(r *SaveProfileRequest) { r.BirthDate = "15/06/1995" },
		"no name":      func(r *SaveProfileRequest) { r.DisplayName = "   " },
		"long name":    func(r *SaveProfileRequest) { r.DisplayName = strings.Repeat("a", 51) },
		"bad gender":   func(r *SaveProfileRequest) { r.Gender = "x" },
		"long bio":     func(r *SaveProfileRequest) { r.Bio = strings.Repeat("a", 501) },
		"control char": func(r *SaveProfileRequest) { r.DisplayName = "bad\x00name" },
		"lat range":    func(r *SaveProfileRequest) { v := 91.0; r.Latitude = &v },
		"half loc":     func(r *SaveProfileRequest) { r.Longitude = nil },
		"bad slug":     func(r *SaveProfileRequest) { r.Interests = []string{"DROP TABLE"} },
		"too many":     func(r *SaveProfileRequest) { r.Interests = strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",") },
	}
	for name, mutate := range cases {
		req := baseRequest()
		mutate(&req)
		_, err := validateProfile(req, fixedNow)
		if _, ok := platformerrors.AsValidation(err); !ok {
			t.Errorf("%s: expected validation error, got %v", name, err)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	good := SavePreferencesRequest{InterestedIn: []string{"man", "man", "woman"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30}
	prefs, err := validatePreferences(good)
	if err != nil || len(prefs.InterestedIn) != 2 {
		t.Fatalf("unexpected result %+v err=%v", prefs, err)
	}
	for name, bad := range map[string]SavePreferencesRequest{
		"empty":    {MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		"minor":    {InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30},
		"inverted": {InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 20, MaxDistanceKm: 30},
		"distance": {InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 0},
		"gender":   {InterestedIn: []string{"robot"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
	} {
		if _, err := validatePreferences(bad); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestApproxDistanceKm(t *testing.T) {
	for in, want := range map[float64]int{0.2: 1, 1: 1, 3.2: 4, 10: 10, 10.1: 15, 47: 50, 100: 100} {
		if got := ApproxDistanceKm(in); got != want {
			t.Errorf("ApproxDistanceKm(%v) = %d, want %d", in, got, want)
		}
	}
}
