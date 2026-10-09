package profiles

import (
	"strings"
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
		{"2008-03-14", 18},
		{"2000-02-29", 26}, // leap day
		{"1990-12-31", 35},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(b, now); got != c.want {
			t.Errorf("AgeOn(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestBucketDistanceNeverExposesPreciseValues(t *testing.T) {
	cases := map[float64]int{0: 1, 0.4: 1, 1: 1, 1.01: 2, 4.9: 5, 5.1: 10, 19.9: 20, 33: 50, 99: 100, 150: 200, 499: 500, 800: 1000}
	for in, want := range cases {
		if got := BucketDistance(in); got != want {
			t.Errorf("BucketDistance(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestRoundCoordinate(t *testing.T) {
	if got := RoundCoordinate(48.85661); got != 48.86 {
		t.Errorf("got %v", got)
	}
	if got := RoundCoordinate(-2.34999); got != -2.35 {
		t.Errorf("got %v", got)
	}
}

func ptr[T any](v T) *T { return &v }

func validRequest() UpdateProfileRequest {
	return UpdateProfileRequest{
		FirstName: "Léa", BirthDate: "1995-06-01", Gender: GenderWoman,
		Latitude: ptr(48.8566), Longitude: ptr(2.3522), Interests: []string{"Music", " music ", "travel"},
	}
}

func TestValidateProfile(t *testing.T) {
	s := &Service{now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}

	in, err := s.validateProfile(validRequest())
	if err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if len(in.InterestSlugs) != 2 || in.InterestSlugs[0] != "music" {
		t.Errorf("interests must be normalised and deduplicated: %v", in.InterestSlugs)
	}
	if in.Latitude != 48.86 || in.Longitude != 2.35 {
		t.Errorf("coordinates must be coarse: %v %v", in.Latitude, in.Longitude)
	}

	mutate := map[string]func(*UpdateProfileRequest){
		"underage":    func(r *UpdateProfileRequest) { r.BirthDate = "2010-06-02" },
		"no coords":   func(r *UpdateProfileRequest) { r.Latitude = nil },
		"lon range":   func(r *UpdateProfileRequest) { r.Longitude = ptr(181.0) },
		"name digits": func(r *UpdateProfileRequest) { r.FirstName = "L3a" },
		"name long":   func(r *UpdateProfileRequest) { r.FirstName = strings.Repeat("a", 41) },
		"bio long":    func(r *UpdateProfileRequest) { r.Bio = strings.Repeat("é", 501) },
		"city long":   func(r *UpdateProfileRequest) { r.City = strings.Repeat("v", 81) },
		"gender":      func(r *UpdateProfileRequest) { r.Gender = "x" },
	}
	for name, f := range mutate {
		req := validRequest()
		f(&req)
		if _, err := s.validateProfile(req); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}

	// Exactly 18 today is allowed; one day short is not.
	req := validRequest()
	req.BirthDate = "2008-01-01"
	if _, err := s.validateProfile(req); err != nil {
		t.Errorf("18th birthday today must be accepted: %v", err)
	}
	req.BirthDate = "2008-01-02"
	if _, err := s.validateProfile(req); err == nil {
		t.Error("17 years and 364 days must be refused")
	}
}

func TestCleanTextStripsControlCharacters(t *testing.T) {
	if got := cleanText("  hi\x00 there\r\nline2\x1b[0m "); got != "hi there\nline2[0m" {
		t.Errorf("got %q", got)
	}
	if got := cleanLine("Paris \n  15e\t"); got != "Paris 15e" {
		t.Errorf("got %q", got)
	}
}
