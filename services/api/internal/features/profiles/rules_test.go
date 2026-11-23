package profiles

import (
	"errors"
	"testing"
	"time"
)

func date(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestAgeOn(t *testing.T) {
	cases := []struct {
		birth, today string
		want         int
	}{
		{"2000-06-15", "2018-06-14", 17},
		{"2000-06-15", "2018-06-15", 18},
		{"2000-02-29", "2019-02-28", 18},
		{"2000-02-29", "2019-03-01", 19},
		{"1990-12-31", "2026-01-01", 35},
	}
	for _, c := range cases {
		if got := AgeOn(date(c.birth), date(c.today)); got != c.want {
			t.Errorf("AgeOn(%s, %s) = %d, want %d", c.birth, c.today, got, c.want)
		}
	}
}

func TestBirthdateBoundsMatchAgeOn(t *testing.T) {
	today := date("2026-11-23")
	oldest, youngest := BirthdateBoundsForAges(25, 30, today)
	if AgeOn(youngest, today) != 25 || AgeOn(youngest.AddDate(0, 0, 1), today) != 24 {
		t.Fatalf("youngest bound wrong: %s", youngest)
	}
	if AgeOn(oldest, today) != 30 || AgeOn(oldest.AddDate(0, 0, -1), today) != 31 {
		t.Fatalf("oldest bound wrong: %s", oldest)
	}
}

func TestParseBirthdateEnforcesAdults(t *testing.T) {
	today := date("2026-11-23")
	if _, err := ParseBirthdate("2008-11-24", today); !errors.Is(err, ErrUnderage) {
		t.Fatalf("expected underage error, got %v", err)
	}
	if _, err := ParseBirthdate("2008-11-23", today); err != nil {
		t.Fatalf("18th birthday today should be allowed: %v", err)
	}
	if _, err := ParseBirthdate("23/11/2000", today); !errors.Is(err, ErrInvalidBirthdate) {
		t.Fatalf("expected invalid format error, got %v", err)
	}
	if _, err := ParseBirthdate("1850-01-01", today); !errors.Is(err, ErrInvalidAge) {
		t.Fatalf("expected implausible age error, got %v", err)
	}
}

func TestPrivacyRounding(t *testing.T) {
	if SnapCoordinate(48.856613) != 48.86 || SnapCoordinate(2.352222) != 2.35 {
		t.Fatal("coordinates must be snapped to 2 decimals")
	}
	if ApproximateDistance(0.2) != 1 || ApproximateDistance(12.6) != 13 {
		t.Fatal("distance must be rounded to whole km with a 1 km floor")
	}
	if ValidateCoordinates(91, 0) == nil || ValidateCoordinates(0, 181) == nil {
		t.Fatal("out of range coordinates must be rejected")
	}
}

func TestNormalizeFirstName(t *testing.T) {
	if got, err := NormalizeFirstName("  Anne-Sophie  "); err != nil || got != "Anne-Sophie" {
		t.Fatalf("got %q %v", got, err)
	}
	if got, err := NormalizeFirstName("Zoé"); err != nil || got != "Zoé" {
		t.Fatalf("got %q %v", got, err)
	}
	for _, bad := range []string{"", "   ", "<script>", "R2D2", "a very long first name that exceeds the limit"} {
		if _, err := NormalizeFirstName(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestNormalizePreferences(t *testing.T) {
	prefs, err := NormalizePreferences(UpdatePreferencesRequest{InterestedIn: []string{"woman", "woman", "nonbinary"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(prefs.InterestedIn) != 2 {
		t.Fatalf("unexpected %v %v", prefs, err)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 301},
	}
	for i, req := range bad {
		if _, err := NormalizePreferences(req); err == nil {
			t.Errorf("case %d: expected rejection", i)
		}
	}
}

func TestNormalizeInterests(t *testing.T) {
	got, err := NormalizeInterests([]string{"Hiking", "hiking", " yoga ", ""})
	if err != nil || len(got) != 2 || got[0] != "hiking" || got[1] != "yoga" {
		t.Fatalf("unexpected %v %v", got, err)
	}
	many := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	if _, err := NormalizeInterests(many); !errors.Is(err, ErrTooManyInterests) {
		t.Fatalf("expected too many interests, got %v", err)
	}
}

func TestMissingFields(t *testing.T) {
	name, gender := "Lou", "woman"
	b := date("1995-01-01")
	lat, lon := 48.85, 2.35
	full := StoredProfile{FirstName: &name, Birthdate: &b, Gender: &gender, Latitude: &lat, Longitude: &lon}
	prefs := Preferences{InterestedIn: []string{"man"}}
	if m := MissingFields(full, prefs, 1); len(m) != 0 {
		t.Fatalf("expected complete profile, missing %v", m)
	}
	if m := MissingFields(StoredProfile{}, Preferences{}, 0); len(m) != 6 {
		t.Fatalf("expected 6 missing fields, got %v", m)
	}
}
