package users

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func valid() ProfileInput {
	return ProfileInput{FirstName: "Léa", BirthDate: "1995-06-15", Gender: GenderWoman, Bio: "Hello", City: "Lyon", InterestIDs: []int{1, 2, 2}}
}

func TestAgeOn(t *testing.T) {
	cases := []struct {
		birth string
		want  int
	}{{"2008-06-15", 18}, {"2008-06-16", 17}, {"1990-12-31", 35}, {"2000-02-29", 26}}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(b, now); got != c.want {
			t.Errorf("%s: got %d want %d", c.birth, got, c.want)
		}
	}
}

func TestValidateProfile(t *testing.T) {
	in, birth, err := ValidateProfile(valid(), now)
	if err != nil || birth.Year() != 1995 || len(in.InterestIDs) != 2 {
		t.Fatalf("valid input rejected or not deduped: %v %+v", err, in)
	}
	bad := map[string]func(*ProfileInput){
		"underage":     func(p *ProfileInput) { p.BirthDate = "2008-06-16" },
		"future":       func(p *ProfileInput) { p.BirthDate = "2030-01-01" },
		"bad date":     func(p *ProfileInput) { p.BirthDate = "15/06/1995" },
		"empty name":   func(p *ProfileInput) { p.FirstName = "  " },
		"digits name":  func(p *ProfileInput) { p.FirstName = "Bob42" },
		"long name":    func(p *ProfileInput) { p.FirstName = strings.Repeat("a", 41) },
		"bad gender":   func(p *ProfileInput) { p.Gender = "x" },
		"long bio":     func(p *ProfileInput) { p.Bio = strings.Repeat("é", 501) },
		"many interes": func(p *ProfileInput) { p.InterestIDs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} },
		"neg interest": func(p *ProfileInput) { p.InterestIDs = []int{-1} },
		"ancient":      func(p *ProfileInput) { p.BirthDate = "1850-01-01" },
	}
	for name, mutate := range bad {
		p := valid()
		mutate(&p)
		if _, _, err := ValidateProfile(p, now); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
	if _, _, err := ValidateProfile(func() ProfileInput { p := valid(); p.BirthDate = "2008-06-16"; return p }(), now); err != ErrUnderage {
		t.Errorf("expected ErrUnderage, got %v", err)
	}
}

func TestValidateProfileStripsControlChars(t *testing.T) {
	p := valid()
	p.Bio = "line1\n\x00line2\r"
	out, _, err := ValidateProfile(p, now)
	if err != nil || out.Bio != "line1\nline2" {
		t.Fatalf("bio = %q err=%v", out.Bio, err)
	}
}

func TestValidatePreferences(t *testing.T) {
	d := 30
	ok := Preferences{InterestedIn: []string{"man", "man", "woman"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: &d}
	got, err := ValidatePreferences(ok)
	if err != nil || len(got.InterestedIn) != 2 {
		t.Fatalf("unexpected: %v %+v", err, got)
	}
	zero, big := 0, 501
	for name, p := range map[string]Preferences{
		"empty":    {InterestedIn: nil, MinAge: 18, MaxAge: 30},
		"unknown":  {InterestedIn: []string{"x"}, MinAge: 18, MaxAge: 30},
		"min<18":   {InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30},
		"max>99":   {InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 100},
		"inverted": {InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30},
		"dist 0":   {InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: &zero},
		"dist big": {InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: &big},
	} {
		if _, err := ValidatePreferences(p); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestApproxDistanceKm(t *testing.T) {
	cases := map[float64]int{0: 1, 0.3: 1, 1.2: 2, 9.9: 10, 11: 15, 50: 50, 51: 60, 389: 390}
	for in, want := range cases {
		if got := ApproxDistanceKm(in); got != want {
			t.Errorf("ApproxDistanceKm(%v) = %d want %d", in, got, want)
		}
	}
}

func TestRoundCoordinate(t *testing.T) {
	cases := map[float64]float64{48.85661: 48.86, -73.9857: -73.99, 2.3522: 2.35, 0: 0}
	for in, want := range cases {
		if got := RoundCoordinate(in); got != want {
			t.Errorf("RoundCoordinate(%v) = %v want %v", in, got, want)
		}
	}
}
