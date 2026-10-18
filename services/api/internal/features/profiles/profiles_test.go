package profiles

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/features/photos"
)

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"2008-06-14", 18},
		{"1990-12-31", 35},
		{"2000-02-29", 26},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(b, now); got != c.want {
			t.Errorf("AgeOn(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestApproxDistanceKmNeverRevealsPreciseValues(t *testing.T) {
	cases := map[float64]int{0.1: 1, 1.9: 1, 2.0: 2, 2.1: 3, 9.4: 10, 10: 10, 10.1: 15, 23: 25, 101: 105}
	for in, want := range cases {
		if got := ApproxDistanceKm(in); got != want {
			t.Errorf("ApproxDistanceKm(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestRoundCoordinate(t *testing.T) {
	if got := RoundCoordinate(45.76423); got != 45.76 {
		t.Fatalf("got %v", got)
	}
	if got := RoundCoordinate(-4.8357); got != -4.84 {
		t.Fatalf("got %v", got)
	}
}

type noPhotos struct{}

func (noPhotos) ListForUsers(context.Context, []string) (map[string][]photos.Photo, error) {
	return nil, nil
}

func f64(v float64) *float64 { return &v }

func validRequest() UpsertRequest {
	return UpsertRequest{
		FirstName: "Amélie-Rose", BirthDate: "1995-04-12", Gender: "woman", Bio: "Salut\x00 !", City: " Lyon  ",
		Latitude: f64(45.7640), Longitude: f64(4.8357), Interests: []string{"Travel", " music ", "travel"},
		Preferences: Preferences{InterestedIn: []string{"man", "man", "woman"}, AgeMin: 25, AgeMax: 40},
	}
}

func TestValidateNormalisesInput(t *testing.T) {
	s := NewService(nil, noPhotos{})
	s.now = func() time.Time { return time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) }
	p, slugs, err := s.validate("u1", validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if p.FirstName != "Amélie-Rose" || p.City != "Lyon" || p.Bio != "Salut !" {
		t.Fatalf("text not normalised: %+v", p)
	}
	if p.Latitude != 45.76 || p.Longitude != 4.84 {
		t.Fatalf("coordinates not rounded: %v,%v", p.Latitude, p.Longitude)
	}
	if len(slugs) != 2 || slugs[0] != "travel" || slugs[1] != "music" {
		t.Fatalf("interests must be lower-cased and de-duplicated: %v", slugs)
	}
	if len(p.Preferences.InterestedIn) != 2 {
		t.Fatalf("genders must be de-duplicated: %v", p.Preferences.InterestedIn)
	}
}

func TestValidateRejectsBadInput(t *testing.T) {
	s := NewService(nil, noPhotos{})
	s.now = func() time.Time { return time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) }
	mutate := func(f func(*UpsertRequest)) UpsertRequest { r := validRequest(); f(&r); return r }

	cases := []struct {
		name string
		req  UpsertRequest
		want error
	}{
		{"underage by one day", mutate(func(r *UpsertRequest) { r.BirthDate = "2008-06-16" }), ErrUnderage},
		{"future birth", mutate(func(r *UpsertRequest) { r.BirthDate = "2030-01-01" }), ErrUnderage},
		{"ancient", mutate(func(r *UpsertRequest) { r.BirthDate = "1800-01-01" }), ErrTooOld},
		{"garbage date", mutate(func(r *UpsertRequest) { r.BirthDate = "yesterday" }), ErrInvalidBirthDate},
		{"html name", mutate(func(r *UpsertRequest) { r.FirstName = "<b>x</b>" }), ErrInvalidName},
		{"digits name", mutate(func(r *UpsertRequest) { r.FirstName = "R2D2" }), ErrInvalidName},
		{"long name", mutate(func(r *UpsertRequest) { r.FirstName = strings.Repeat("a", 41) }), ErrInvalidName},
		{"long bio", mutate(func(r *UpsertRequest) { r.Bio = strings.Repeat("é", 501) }), ErrBioTooLong},
		{"long city", mutate(func(r *UpsertRequest) { r.City = strings.Repeat("v", 81) }), ErrCityTooLong},
		{"inverted ages", mutate(func(r *UpsertRequest) { r.Preferences.AgeMin, r.Preferences.AgeMax = 40, 30 }), ErrInvalidPrefs},
		{"minor preference", mutate(func(r *UpsertRequest) { r.Preferences.AgeMin = 16 }), ErrInvalidPrefs},
		{"too many interests", mutate(func(r *UpsertRequest) {
			r.Interests = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
		}), ErrTooManyInterests},
	}
	for _, c := range cases {
		if _, _, err := s.validate("u1", c.req); !errors.Is(err, c.want) {
			t.Errorf("%s: want %v, got %v", c.name, c.want, err)
		}
	}
}

func TestHydrateHidesUnknownDistanceAndBuildsEmptyLists(t *testing.T) {
	s := NewService(&stubRepo{}, noPhotos{})
	s.now = func() time.Time { return time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) }
	km := 12.3
	out, err := s.Hydrate(context.Background(), []Card{
		{UserID: "a", FirstName: "A", BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), DistanceKm: &km},
		{UserID: "b", FirstName: "B", BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].DistanceKm == nil || *out[0].DistanceKm != 15 {
		t.Fatalf("distance must be bucketed: %v", out[0].DistanceKm)
	}
	if out[1].DistanceKm != nil || out[1].Interests == nil || out[1].Photos == nil {
		t.Fatalf("hidden distance must stay nil and lists non-nil: %+v", out[1])
	}
}

type stubRepo struct{ Repository }

func (*stubRepo) InterestsForUsers(context.Context, []string) (map[string][]InterestRef, error) {
	return map[string][]InterestRef{}, nil
}
