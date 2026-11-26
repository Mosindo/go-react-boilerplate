package profiles

import (
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestAgeOn(t *testing.T) {
	birth := date(2000, time.June, 15)
	cases := []struct {
		on   time.Time
		want int
	}{
		{date(2018, time.June, 14), 17},
		{date(2018, time.June, 15), 18},
		{date(2026, time.January, 1), 25},
	}
	for _, c := range cases {
		if got := AgeOn(birth, c.on); got != c.want {
			t.Errorf("AgeOn(%v) = %d, want %d", c.on, got, c.want)
		}
	}
}

func TestParseBirthdateEnforcesAdults(t *testing.T) {
	now := today()
	adult := now.AddDate(-18, 0, 0).Format(birthdateLayout)
	minor := now.AddDate(-18, 0, 1).Format(birthdateLayout)
	if _, err := ParseBirthdate(adult); err != nil {
		t.Fatalf("18 today must be accepted: %v", err)
	}
	if _, err := ParseBirthdate(minor); err == nil {
		t.Fatal("17 must be rejected")
	}
	for _, bad := range []string{"", "1990-13-01", "01/02/1990", "1900-01-01"} {
		if _, err := ParseBirthdate(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestNormalizeFirstName(t *testing.T) {
	got, err := NormalizeFirstName("  Anne-Marie   d'Arc ")
	if err != nil || got != "Anne-Marie d'Arc" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "   ", "<script>", "Bob123", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		if _, err := NormalizeFirstName(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	ok, err := ValidatePreferences(UpdatePreferencesRequest{InterestedIn: []string{"woman", "woman", "man"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(ok.InterestedIn) != 2 {
		t.Fatalf("expected deduplicated preferences, got %+v %v", ok, err)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"robot"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 0},
	}
	for i, req := range bad {
		if _, err := ValidatePreferences(req); err == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
}

func TestLocationPrivacy(t *testing.T) {
	if got := CoarsenCoordinate(48.856613); got != 48.86 {
		t.Fatalf("coarsen = %v", got)
	}
	// Paris -> Lyon is ~392 km.
	d := HaversineKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("unexpected distance %v", d)
	}
	cases := map[float64]int{0.2: 1, 3.4: 3, 12.4: 10, 13: 15, 391.7: 390}
	for in, want := range cases {
		if got := ApproxDistanceKm(in); got != want {
			t.Errorf("ApproxDistanceKm(%v) = %d, want %d", in, got, want)
		}
	}
}
