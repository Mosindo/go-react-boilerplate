package profiles

import (
	"strings"
	"testing"
	"time"
)

func TestValidateBirthDateEnforcesMajority(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		in string
		ok bool
	}{
		{"2008-06-15", true},  // turns 18 today
		{"2008-06-16", false}, // one day short
		{"2010-01-01", false},
		{"1990-02-30", false}, // impossible date
		{"15/06/1990", false},
		{"1890-01-01", false},
		{"1990-05-05", true},
	}
	for _, c := range cases {
		_, err := ValidateBirthDate(c.in, now)
		if (err == nil) != c.ok {
			t.Errorf("ValidateBirthDate(%q) err=%v, want ok=%v", c.in, err, c.ok)
		}
	}
}

func TestAgeOnLeapDay(t *testing.T) {
	born := time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC)
	if got := AgeOn(born, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)); got != 25 {
		t.Fatalf("expected 25 the day before, got %d", got)
	}
	if got := AgeOn(born, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)); got != 26 {
		t.Fatalf("expected 26, got %d", got)
	}
}

func TestValidateName(t *testing.T) {
	if n, err := ValidateName("  Anne-Laure   Dupont "); err != nil || n != "Anne-Laure Dupont" {
		t.Fatalf("unexpected: %q %v", n, err)
	}
	for _, bad := range []string{"", "   ", "<script>", "Bob123", strings.Repeat("a", 41)} {
		if _, err := ValidateName(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestValidateBioStripsControlCharacters(t *testing.T) {
	bio, err := ValidateBio("hello\u0000 wor​ld\nline2")
	if err != nil || bio != "hello world\nline2" {
		t.Fatalf("unexpected bio %q err=%v", bio, err)
	}
	if _, err := ValidateBio(strings.Repeat("é", 501)); err == nil {
		t.Fatal("expected overlong bio to be rejected")
	}
}

func TestValidatePreferences(t *testing.T) {
	dist := 30
	ok, err := ValidatePreferences(PreferencesRequest{InterestedIn: []string{"Woman", "woman", "man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: &dist})
	if err != nil || len(ok.InterestedIn) != 2 {
		t.Fatalf("unexpected: %+v %v", ok, err)
	}
	bad := []PreferencesRequest{
		{InterestedIn: nil, MinAge: 20, MaxAge: 30},
		{InterestedIn: []string{"alien"}, MinAge: 20, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 100},
	}
	for i, b := range bad {
		if _, err := ValidatePreferences(b); err == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
	zero := 0
	if _, err := ValidatePreferences(PreferencesRequest{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: &zero}); err == nil {
		t.Error("distance 0 should be rejected")
	}
}

func TestLocationIsRoundedAndValidated(t *testing.T) {
	la, lo := 48.856614, 2.352222
	gotLa, gotLo, err := ValidateLocation(LocationRequest{Latitude: &la, Longitude: &lo})
	if err != nil || *gotLa != 48.86 || *gotLo != 2.35 {
		t.Fatalf("unexpected rounding: %v %v %v", gotLa, gotLo, err)
	}
	bad := 91.0
	if _, _, err := ValidateLocation(LocationRequest{Latitude: &bad, Longitude: &lo}); err == nil {
		t.Fatal("expected out of range latitude to fail")
	}
	if _, _, err := ValidateLocation(LocationRequest{Latitude: &la}); err == nil {
		t.Fatal("expected half coordinate to fail")
	}
	if la, lo, err := ValidateLocation(LocationRequest{}); err != nil || la != nil || lo != nil {
		t.Fatal("empty location clears coordinates")
	}
}

func TestDistanceBucket(t *testing.T) {
	cases := map[float64]int{0: 5, 0.4: 5, 5: 5, 5.1: 10, 12: 15, 99.9: 100}
	for in, want := range cases {
		if got := DistanceBucket(in); got != want {
			t.Errorf("DistanceBucket(%v)=%d want %d", in, got, want)
		}
	}
}

func TestValidateInterestIDs(t *testing.T) {
	ids, err := ValidateInterestIDs([]int{1, 2, 2, 3})
	if err != nil || len(ids) != 3 {
		t.Fatalf("unexpected %v %v", ids, err)
	}
	many := make([]int, 11)
	for i := range many {
		many[i] = i + 1
	}
	if _, err := ValidateInterestIDs(many); err == nil {
		t.Fatal("expected >10 interests to fail")
	}
	if _, err := ValidateInterestIDs([]int{0}); err == nil {
		t.Fatal("expected id 0 to fail")
	}
}
