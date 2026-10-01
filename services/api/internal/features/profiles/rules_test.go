package profiles

import (
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, time.UTC) }

func TestAgeOnBirthdayBoundary(t *testing.T) {
	now := d(2026, 6, 15)
	cases := []struct {
		birth time.Time
		want  int
	}{
		{d(2008, 6, 15), 18},
		{d(2008, 6, 16), 17},
		{d(2000, 12, 31), 25},
		{d(2000, 2, 29), 26},
	}
	for _, c := range cases {
		if got := AgeOn(c.birth, now); got != c.want {
			t.Errorf("AgeOn(%v)=%d want %d", c.birth, got, c.want)
		}
	}
}

func TestValidateBirthDate(t *testing.T) {
	now := d(2026, 6, 15)
	if err := ValidateBirthDate(d(2008, 6, 16), now); err != ErrUnderage {
		t.Errorf("17y364d should be underage, got %v", err)
	}
	if err := ValidateBirthDate(d(2008, 6, 15), now); err != nil {
		t.Errorf("exactly 18 should pass, got %v", err)
	}
	if err := ValidateBirthDate(d(2030, 1, 1), now); err != ErrInvalidBirthDate {
		t.Errorf("future date should be invalid, got %v", err)
	}
	if err := ValidateBirthDate(d(1800, 1, 1), now); err != ErrInvalidBirthDate {
		t.Errorf("too old should be invalid, got %v", err)
	}
}

func TestBucketDistance(t *testing.T) {
	cases := map[float64]int{0: 5, 0.2: 5, 5: 5, 5.1: 10, 12: 15, 49.9: 50, 50: 50, 50.01: 55}
	for in, want := range cases {
		if got := BucketDistanceKm(in); got != want {
			t.Errorf("BucketDistanceKm(%v)=%d want %d", in, got, want)
		}
	}
}

func TestSnapCoordinate(t *testing.T) {
	if got := SnapCoordinate(48.85661); got != 48.86 {
		t.Errorf("got %v", got)
	}
	if got := SnapCoordinate(-2.34999); got != -2.35 {
		t.Errorf("got %v", got)
	}
}

func TestValidateLocation(t *testing.T) {
	if ValidateLocation(91, 0) == nil || ValidateLocation(0, 181) == nil {
		t.Error("out of range should fail")
	}
	if ValidateLocation(48.8, 2.3) != nil {
		t.Error("valid should pass")
	}
}

func TestValidatePreferences(t *testing.T) {
	ok := Preferences{InterestedIn: []string{"man", "woman"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30}
	if err := ValidatePreferences(ok); err != nil {
		t.Fatal(err)
	}
	bad := []Preferences{
		{InterestedIn: nil, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"x"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man", "man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 501},
	}
	for i, p := range bad {
		if ValidatePreferences(p) == nil {
			t.Errorf("case %d should fail", i)
		}
	}
}

func TestCleanTextAndLimits(t *testing.T) {
	if got := CleanText("  hé\x00llo\x07\nmonde  "); got != "héllo\nmonde" {
		t.Errorf("got %q", got)
	}
	if ValidateProfileText("", "", "") != ErrInvalidName {
		t.Error("empty name")
	}
	long := make([]rune, 501)
	for i := range long {
		long[i] = 'é'
	}
	if ValidateProfileText("A", string(long), "") != ErrBioTooLong {
		t.Error("bio limit counts runes")
	}
}
