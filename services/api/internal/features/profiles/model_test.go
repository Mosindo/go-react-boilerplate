package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func TestAgeAt(t *testing.T) {
	birth := time.Date(2000, 6, 16, 0, 0, 0, 0, time.UTC)
	if got := AgeAt(birth, fixedNow); got != 25 { // anniversaire le lendemain
		t.Fatalf("got %d", got)
	}
	if got := AgeAt(time.Date(2000, 6, 15, 0, 0, 0, 0, time.UTC), fixedNow); got != 26 {
		t.Fatalf("birthday today: %d", got)
	}
}

func TestNormalizeAgeRule(t *testing.T) {
	ok := ProfileInput{FirstName: " Léa ", BirthDate: "2008-06-15", Gender: "Woman"}
	if _, err := ok.Normalize(fixedNow); err != nil {
		t.Fatalf("exactly 18 must be accepted: %v", err)
	}
	if ok.FirstName != "Léa" || ok.Gender != "woman" {
		t.Fatalf("not normalized: %+v", ok)
	}
	minor := ProfileInput{FirstName: "Tim", BirthDate: "2008-06-16", Gender: "man"}
	if _, err := minor.Normalize(fixedNow); !errors.Is(err, ErrUnderage) {
		t.Fatalf("17 years old must be rejected, got %v", err)
	}
}

func TestNormalizeRejectsBadInput(t *testing.T) {
	cases := map[string]ProfileInput{
		"control char": {FirstName: "A\x00", BirthDate: "1990-01-01", Gender: "man"},
		"long name":    {FirstName: strings.Repeat("a", 41), BirthDate: "1990-01-01", Gender: "man"},
		"too old":      {FirstName: "A", BirthDate: "1900-01-01", Gender: "man"},
		"many interests": {FirstName: "A", BirthDate: "1990-01-01", Gender: "man",
			Interests: strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",")},
	}
	for name, in := range cases {
		if _, err := in.Normalize(fixedNow); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	dup := ProfileInput{FirstName: "A", BirthDate: "1990-01-01", Gender: "man", Interests: []string{"Music", "music", " music "}}
	if _, err := dup.Normalize(fixedNow); err != nil || len(dup.Interests) != 1 {
		t.Fatalf("interests should be deduplicated: %v %v", err, dup.Interests)
	}
}

func TestSettingsValidate(t *testing.T) {
	s := DefaultSettings()
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []Settings{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 501},
	}
	for i, b := range bad {
		if err := b.Validate(); err == nil {
			t.Errorf("case %d should fail", i)
		}
	}
}
