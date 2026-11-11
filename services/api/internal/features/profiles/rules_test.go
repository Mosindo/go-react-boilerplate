package profiles

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	birth := time.Date(2000, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		now  time.Time
		want int
	}{
		{time.Date(2018, 6, 14, 0, 0, 0, 0, time.UTC), 17},
		{time.Date(2018, 6, 15, 0, 0, 0, 0, time.UTC), 18},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 25},
	}
	for _, c := range cases {
		if got := AgeOn(birth, c.now); got != c.want {
			t.Errorf("AgeOn(%v) = %d, want %d", c.now, got, c.want)
		}
	}
}

func TestRoundCoordinateAndDistance(t *testing.T) {
	if got := RoundCoordinate(48.856613); got != 48.86 {
		t.Fatalf("RoundCoordinate = %v", got)
	}
	// Paris -> Lyon is ~392 km.
	d := HaversineKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("unexpected Paris-Lyon distance %v", d)
	}
	if DisplayDistanceKm(0.3) != 1 || DisplayDistanceKm(7.4) != 7 || DisplayDistanceKm(392) != 390 {
		t.Fatal("unexpected display rounding")
	}
}

func TestValidateProfileInput(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ok := ProfileRequest{FirstName: "  Camille ", Birthdate: "1995-04-02", Gender: "woman", Bio: "Salut"}
	p, err := validateProfile("u1", ok, now)
	if err != nil || p.FirstName != "Camille" {
		t.Fatalf("expected valid profile, got %v %#v", err, p)
	}

	minor := ok
	minor.Birthdate = "2010-01-01"
	if _, err := validateProfile("u1", minor, now); err != ErrUnderage {
		t.Fatalf("expected ErrUnderage, got %v", err)
	}
	badGender := ok
	badGender.Gender = "robot"
	if _, err := validateProfile("u1", badGender, now); err != ErrInvalidGender {
		t.Fatalf("expected ErrInvalidGender, got %v", err)
	}
	badDate := ok
	badDate.Birthdate = "1995-13-40"
	if _, err := validateProfile("u1", badDate, now); err != ErrInvalidBirthdate {
		t.Fatalf("expected ErrInvalidBirthdate, got %v", err)
	}
	noName := ok
	noName.FirstName = "   "
	if _, err := validateProfile("u1", noName, now); err != ErrInvalidFirstName {
		t.Fatalf("expected ErrInvalidFirstName, got %v", err)
	}
}

func TestValidatePreferences(t *testing.T) {
	good := PreferencesRequest{InterestedIn: []string{"man", "woman", "man"}, AgeMin: 25, AgeMax: 40, MaxDistanceKm: 30}
	p, err := validatePreferences(good)
	if err != nil || len(p.InterestedIn) != 2 {
		t.Fatalf("expected dedup + valid, got %v %#v", err, p)
	}
	for _, bad := range []PreferencesRequest{
		{InterestedIn: []string{"x"}, AgeMin: 25, AgeMax: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, AgeMin: 17, AgeMax: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, AgeMin: 40, AgeMax: 25, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, AgeMin: 25, AgeMax: 40, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, AgeMin: 25, AgeMax: 40, MaxDistanceKm: 501},
	} {
		if _, err := validatePreferences(bad); err != ErrInvalidPreferences {
			t.Errorf("expected ErrInvalidPreferences for %#v, got %v", bad, err)
		}
	}
}
