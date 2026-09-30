package profiles

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	birth := time.Date(2000, 6, 15, 0, 0, 0, 0, time.UTC)
	if got := AgeOn(birth, time.Date(2018, 6, 14, 0, 0, 0, 0, time.UTC)); got != 17 {
		t.Fatalf("day before birthday: got %d", got)
	}
	if got := AgeOn(birth, time.Date(2018, 6, 15, 0, 0, 0, 0, time.UTC)); got != 18 {
		t.Fatalf("on birthday: got %d", got)
	}
}

func TestParseBirthdateEnforcesAdults(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	if _, err := ParseBirthdate("2008-01-11", now); err == nil {
		t.Fatal("17-year-old must be rejected")
	}
	if _, err := ParseBirthdate("2008-01-10", now); err != nil {
		t.Fatalf("18-year-old must be accepted: %v", err)
	}
	if _, err := ParseBirthdate("10/01/1990", now); err == nil {
		t.Fatal("wrong format must be rejected")
	}
	if _, err := ParseBirthdate("1900-01-01", now); err == nil {
		t.Fatal("implausible age must be rejected")
	}
}

func TestValidateFirstName(t *testing.T) {
	for _, ok := range []string{"Élodie", "Jean-Luc", "O'Neil", "  Mia  "} {
		if _, err := ValidateFirstName(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "   ", "<script>", "Bob123", "https://spam.example"} {
		if _, err := ValidateFirstName(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestCleanTextLimitsAndControlChars(t *testing.T) {
	got, err := CleanText("  hello\x00world\n\n\n\nbye ", 50, "bio", true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world\n\nbye" {
		t.Fatalf("unexpected cleaned text %q", got)
	}
	if _, err := CleanText("ééééé", 4, "bio", false); err == nil {
		t.Fatal("rune limit must be enforced")
	}
}

func TestValidatePreferences(t *testing.T) {
	prefs, err := ValidatePreferences(UpdatePreferencesRequest{InterestedIn: []string{"woman", "woman", "man"}, MinAge: 25, MaxAge: 35, MaxDistanceKm: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(prefs.InterestedIn) != 2 {
		t.Fatalf("duplicates must be removed: %v", prefs.InterestedIn)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 900},
	}
	for i, req := range bad {
		if _, err := ValidatePreferences(req); err == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
}

func TestComputeCompleteness(t *testing.T) {
	gender := "woman"
	birth := time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)
	p := storedProfile{FirstName: "Ana", Birthdate: &birth, Gender: &gender, Preferences: Preferences{InterestedIn: []string{"man"}}}
	if c := ComputeCompleteness(p, 0); c.Complete || len(c.Missing) != 1 || c.Missing[0] != "photos" {
		t.Fatalf("expected only photos missing, got %+v", c)
	}
	if c := ComputeCompleteness(p, 1); !c.Complete {
		t.Fatalf("expected complete, got %+v", c)
	}
}
