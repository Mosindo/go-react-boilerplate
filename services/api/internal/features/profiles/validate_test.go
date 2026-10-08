package profiles

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-03-15", 18}, // birthday today
		{"2008-03-16", 17}, // birthday tomorrow
		{"2000-02-29", 26}, // leap day
		{"1990-12-31", 35},
	}
	for _, c := range cases {
		bd, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(bd, now); got != c.want {
			t.Errorf("AgeOn(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestValidateProfile(t *testing.T) {
	now := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	ok := UpdateProfileRequest{FirstName: "  Jean   Pierre ", Birthdate: "1995-06-01", Gender: "man", Bio: "  hi\x00 there  ", Interests: []string{"travel", " travel ", ""}}
	v, err := validateProfile(ok, now)
	if err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	if v.FirstName != "Jean Pierre" || v.Bio != "hi there" || len(v.Interests) != 1 || v.LookingFor != "open" {
		t.Fatalf("not normalised: %+v", v)
	}

	bad := map[string]UpdateProfileRequest{
		"18th birthday tomorrow": {FirstName: "A", Birthdate: "2008-03-16", Gender: "man"},
		"html in name":           {FirstName: "<b>x</b>", Birthdate: "1995-06-01", Gender: "man"},
		"emoji name":             {FirstName: "Bob😀", Birthdate: "1995-06-01", Gender: "man"},
		"future":                 {FirstName: "A", Birthdate: "2030-01-01", Gender: "man"},
	}
	for name, req := range bad {
		if _, err := validateProfile(req, now); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if _, err := validateProfile(UpdateProfileRequest{FirstName: "Zoë", Birthdate: "2008-03-15", Gender: "woman"}, now); err != nil {
		t.Errorf("exactly 18 today must be accepted: %v", err)
	}
}

func TestDisplayDistanceBuckets(t *testing.T) {
	for km, want := range map[float64]int{0: 5, 0.4: 5, 5: 5, 5.1: 10, 12: 15, 99.9: 100} {
		if got := DisplayDistance(km); got != want {
			t.Errorf("DisplayDistance(%v) = %d, want %d", km, got, want)
		}
	}
}

func TestSnapCoordinate(t *testing.T) {
	if got := snapCoordinate(48.856614); got != 48.86 {
		t.Fatalf("got %v", got)
	}
	if got := snapCoordinate(-0.004); got != 0 {
		t.Fatalf("got %v", got)
	}
}
