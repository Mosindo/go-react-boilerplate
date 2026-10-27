package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

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

func newTestService() *Service {
	s := NewService(nil, nil)
	s.now = func() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) }
	return s
}

func validRequest() UpsertProfileRequest {
	return UpsertProfileRequest{FirstName: "  Zoé  ", BirthDate: "1995-03-10", Gender: "woman", Bio: " hi ", City: "Lyon", Interests: []string{"Music", "music", " hiking "}}
}

func TestValidateProfileNormalizes(t *testing.T) {
	in, err := newTestService().validateProfile(validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if in.FirstName != "Zoé" || in.Bio != "hi" || len(in.Interests) != 2 || in.Interests[0] != "music" {
		t.Fatalf("not normalized: %+v", in)
	}
}

func TestValidateProfileRejects(t *testing.T) {
	s := newTestService()
	lat, lng := 91.0, 10.0
	one := 10.0
	cases := map[string]func(*UpsertProfileRequest){
		"empty name":       func(r *UpsertProfileRequest) { r.FirstName = "   " },
		"digits in name":   func(r *UpsertProfileRequest) { r.FirstName = "R2D2" },
		"name too long":    func(r *UpsertProfileRequest) { r.FirstName = strings.Repeat("a", 41) },
		"bad date":         func(r *UpsertProfileRequest) { r.BirthDate = "10/03/1995" },
		"bad gender":       func(r *UpsertProfileRequest) { r.Gender = "x" },
		"bio too long":     func(r *UpsertProfileRequest) { r.Bio = strings.Repeat("é", 501) },
		"control in bio":   func(r *UpsertProfileRequest) { r.Bio = "a\x00b" },
		"lat out of range": func(r *UpsertProfileRequest) { r.Latitude, r.Longitude = &lat, &lng },
		"lat without lng":  func(r *UpsertProfileRequest) { r.Latitude = &one },
		"too many interests": func(r *UpsertProfileRequest) {
			r.Interests = strings.Split("a b c d e f g h i j k", " ")
		},
		"too old": func(r *UpsertProfileRequest) { r.BirthDate = "1900-01-01" },
	}
	for name, mutate := range cases {
		req := validRequest()
		mutate(&req)
		if _, err := s.validateProfile(req); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	req := validRequest()
	req.BirthDate = "2010-01-01"
	if _, err := s.validateProfile(req); !errors.Is(err, ErrUnderage) {
		t.Errorf("underage: got %v", err)
	}
}

func TestCoordinatesAreCoarsened(t *testing.T) {
	req := validRequest()
	lat, lng := 48.856614, 2.352222
	req.Latitude, req.Longitude = &lat, &lng
	in, err := newTestService().validateProfile(req)
	if err != nil {
		t.Fatal(err)
	}
	if *in.Latitude != 48.86 || *in.Longitude != 2.35 {
		t.Fatalf("coordinates must be rounded to ~1 km, got %v,%v", *in.Latitude, *in.Longitude)
	}
}
