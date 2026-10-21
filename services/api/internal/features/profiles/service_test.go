package profiles

import (
	"context"
	"strings"
	"testing"
	"time"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
)

type fakeRepo struct{ Repository }

func (fakeRepo) ListInterests(context.Context) ([]Interest, error) {
	return []Interest{{Slug: "travel"}, {Slug: "music"}}, nil
}

func newSvc() *Service {
	return NewService(fakeRepo{}, storage.NewSigner([]byte("profiles-secret-0123456789-abcdefghijk")))
}

func valid() ProfileInput {
	return ProfileInput{FirstName: "Ana", Gender: "woman", LookingFor: "long_term", Interests: []string{"travel"}}
}

func TestNormalizeAcceptsValidProfile(t *testing.T) {
	in := valid()
	in.FirstName = "  Ana \n Maria\t"
	in.City = "  Paris "
	if err := newSvc().normalize(context.Background(), &in); err != nil {
		t.Fatal(err)
	}
	if in.FirstName != "Ana Maria" || in.City != "Paris" {
		t.Fatalf("fields must be cleaned: %q %q", in.FirstName, in.City)
	}
}

func TestNormalizeRejectsInvalid(t *testing.T) {
	h := 20
	mut := map[string]func(*ProfileInput){
		"empty name":     func(p *ProfileInput) { p.FirstName = "  \n " },
		"long name":      func(p *ProfileInput) { p.FirstName = strings.Repeat("a", 41) },
		"gender":         func(p *ProfileInput) { p.Gender = "robot" },
		"long bio":       func(p *ProfileInput) { p.Bio = strings.Repeat("a", 501) },
		"height":         func(p *ProfileInput) { p.HeightCM = &h },
		"looking for":    func(p *ProfileInput) { p.LookingFor = "marriage-tomorrow" },
		"unknown intr.":  func(p *ProfileInput) { p.Interests = []string{"travel", "skydiving"} },
		"too many intr.": func(p *ProfileInput) { p.Interests = strings.Split("a b c d e f g h i j k", " ") },
	}
	for name, f := range mut {
		in := valid()
		f(&in)
		err := newSvc().normalize(context.Background(), &in)
		if be, ok := perr.As(err); !ok || be.Kind != perr.KindInvalid {
			t.Errorf("%s: expected a validation error, got %v", name, err)
		}
	}
}

func TestSavePreferencesValidation(t *testing.T) {
	svc := newSvc()
	d := 600
	bad := []Preferences{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30},
		{InterestedIn: []string{"x"}, MinAge: 18, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 120},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: &d},
	}
	for i, p := range bad {
		if _, err := svc.SavePreferences(context.Background(), "u", p); err == nil {
			t.Errorf("preferences #%d must be rejected", i)
		}
	}
}

func TestAgeOnAndUUID(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	if ageOn(time.Date(2000, 1, 11, 0, 0, 0, 0, time.UTC), now) != 25 {
		t.Fatal("birthday tomorrow: still 25")
	}
	if !validUUID("123e4567-e89b-12d3-a456-426614174000") || validUUID("not-a-uuid") || validUUID("123e4567-e89b-12d3-a456-42661417400g") {
		t.Fatal("uuid validation broken")
	}
}
