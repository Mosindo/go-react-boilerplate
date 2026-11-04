package profiles

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	today := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, {"2008-06-16", 17}, {"1990-01-01", 36}, {"2000-12-31", 25},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(b, today); got != c.want {
			t.Errorf("AgeOn(%s)=%d want %d", c.birth, got, c.want)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	ok := Preferences{InterestedIn: []string{"woman", "woman"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 20}
	got, err := ValidatePreferences(ok)
	if err != nil || len(got.InterestedIn) != 1 {
		t.Fatalf("valid prefs rejected or not deduped: %v %v", got, err)
	}
	bad := []Preferences{
		{InterestedIn: nil, MinAge: 20, MaxAge: 30, MaxDistanceKm: 20},
		{InterestedIn: []string{"alien"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 20},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 20},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 20},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 501},
	}
	for i, p := range bad {
		if _, err := ValidatePreferences(p); err == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
}

func TestValidateName(t *testing.T) {
	if n, err := validateName("  Léa   Marie "); err != nil || n != "Léa Marie" {
		t.Fatalf("got %q %v", n, err)
	}
	if _, err := validateName("   "); err == nil {
		t.Fatal("blank name accepted")
	}
}
