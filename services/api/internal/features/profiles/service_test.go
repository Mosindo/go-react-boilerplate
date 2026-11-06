package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func fixedService() *Service {
	s := NewService(nil)
	s.now = func() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) }
	return s
}

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"1990-12-31", 35},
		{"2000-02-29", 26},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeAt(b, now); got != c.want {
			t.Errorf("AgeAt(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestDistanceBucketAndRounding(t *testing.T) {
	for km, want := range map[float64]int{0: 5, 0.2: 5, 5: 5, 5.1: 10, 12: 15, 99.9: 100} {
		if got := DistanceBucket(km); got != want {
			t.Errorf("DistanceBucket(%v) = %d, want %d", km, got, want)
		}
	}
	if got := RoundCoordinate(48.856614); got != 48.86 {
		t.Errorf("RoundCoordinate = %v", got)
	}
}

func valid() ProfileInput {
	return ProfileInput{FirstName: "  Anne-Sophie ", Gender: "woman", Bio: "Salut\x00 !", City: "Paris", Latitude: 48.856, Longitude: 2.352, Interests: []string{"Travel", "travel", " music "}}
}

func TestValidateProfile(t *testing.T) {
	s := fixedService()
	in, err := s.ValidateProfile(valid(), "1995-04-02")
	if err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	if in.FirstName != "Anne-Sophie" || in.Bio != "Salut !" || in.Latitude != 48.86 || len(in.Interests) != 2 {
		t.Fatalf("unexpected normalisation: %+v", in)
	}

	mutations := map[string]func(*ProfileInput, *string){
		"minor":        func(_ *ProfileInput, b *string) { *b = "2009-01-01" },
		"too old":      func(_ *ProfileInput, b *string) { *b = "1900-01-01" },
		"future":       func(_ *ProfileInput, b *string) { *b = "2030-01-01" },
		"bad format":   func(_ *ProfileInput, b *string) { *b = "01/02/1990" },
		"empty name":   func(p *ProfileInput, _ *string) { p.FirstName = "   " },
		"digit name":   func(p *ProfileInput, _ *string) { p.FirstName = "R2D2" },
		"gender":       func(p *ProfileInput, _ *string) { p.Gender = "other" },
		"long bio":     func(p *ProfileInput, _ *string) { p.Bio = strings.Repeat("é", 501) },
		"lat":          func(p *ProfileInput, _ *string) { p.Latitude = 91 },
		"lng":          func(p *ProfileInput, _ *string) { p.Longitude = -181 },
		"too many int": func(p *ProfileInput, _ *string) { p.Interests = strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",") },
	}
	for name, mutate := range mutations {
		p, birth := valid(), "1995-04-02"
		mutate(&p, &birth)
		if _, err := s.ValidateProfile(p, birth); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: expected validation error, got %v", name, err)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	s := fixedService()
	p, err := s.ValidatePreferences(Preferences{InterestedIn: []string{"Woman", "man", "woman"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(p.InterestedIn) != 2 {
		t.Fatalf("unexpected: %+v %v", p, err)
	}
	bad := []Preferences{
		{InterestedIn: nil, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"x"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 30, MaxAge: 20, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 100, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 501},
	}
	for i, b := range bad {
		if _, err := s.ValidatePreferences(b); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("case %d: expected error", i)
		}
	}
}
