package profiles

import (
	"strings"
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"2000-01-01", 26},
		{"2000-02-29", 26}, // leap day
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(b, now); got != c.want {
			t.Errorf("AgeOn(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestValidateProfile(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	ok := UpsertProfileRequest{FirstName: " Zoé ", BirthDate: "2000-01-01", Gender: "woman", Bio: " hi ", Latitude: fp(48.85661), Longitude: fp(2.35222)}
	v, err := validateProfile(ok, now)
	if err != nil {
		t.Fatal(err)
	}
	if v.FirstName != "Zoé" || v.Bio != "hi" {
		t.Errorf("text must be trimmed: %+v", v)
	}
	if *v.Latitude != 48.86 || *v.Longitude != 2.35 {
		t.Errorf("coordinates must be rounded to 0.01°: %v %v", *v.Latitude, *v.Longitude)
	}
	if !v.ShowDistance || !v.Discoverable {
		t.Error("defaults: distance shown and discoverable")
	}

	bad := map[string]func(*UpsertProfileRequest){
		"underage by one day": func(r *UpsertProfileRequest) { r.BirthDate = "2008-06-16" },
		"too old":             func(r *UpsertProfileRequest) { r.BirthDate = "1900-01-01" },
		"bad date":            func(r *UpsertProfileRequest) { r.BirthDate = "01/01/2000" },
		"control char name":   func(r *UpsertProfileRequest) { r.FirstName = "a\x00b" },
		"newline in name":     func(r *UpsertProfileRequest) { r.FirstName = "a\nb" },
		"long name":           func(r *UpsertProfileRequest) { r.FirstName = strings.Repeat("a", 41) },
		"gender":              func(r *UpsertProfileRequest) { r.Gender = "x" },
		"long bio":            func(r *UpsertProfileRequest) { r.Bio = strings.Repeat("é", 501) },
		"lat range":           func(r *UpsertProfileRequest) { r.Latitude = fp(91) },
		"lng missing":         func(r *UpsertProfileRequest) { r.Longitude = nil },
		"interest negative":   func(r *UpsertProfileRequest) { r.InterestIDs = []int{-1} },
		"too many interests":  func(r *UpsertProfileRequest) { r.InterestIDs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} },
	}
	for name, mut := range bad {
		r := ok
		mut(&r)
		if _, err := validateProfile(r, now); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	// Exactly 18 today is accepted.
	r := ok
	r.BirthDate = "2008-06-15"
	if _, err := validateProfile(r, now); err != nil {
		t.Errorf("18th birthday must be accepted: %v", err)
	}
}

func TestValidatePreferences(t *testing.T) {
	good := UpdatePreferencesRequest{InterestedIn: []string{"man", "woman"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30}
	if _, err := validatePreferences(good); err != nil {
		t.Fatal(err)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man", "man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"alien"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 501},
	}
	for i, b := range bad {
		if _, err := validatePreferences(b); err == nil {
			t.Errorf("case %d must be rejected", i)
		}
	}
}

func fp(v float64) *float64 { return &v }
