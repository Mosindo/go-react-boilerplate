package profiles

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	saved   *ProfileInput
	prefs   *Preferences
	unknown bool
}

func (f *fakeRepo) ListInterests(context.Context) ([]Interest, error)              { return nil, nil }
func (f *fakeRepo) Overview(context.Context, string) (Overview, error)             { return Overview{}, nil }
func (f *fakeRepo) UpdatePrivacy(context.Context, string, Privacy) error           { return nil }
func (f *fakeRepo) UpdateLocation(context.Context, string, float64, float64) error { return nil }
func (f *fakeRepo) ClearLocation(context.Context, string) error                    { return nil }
func (f *fakeRepo) UpsertProfile(_ context.Context, _ string, in ProfileInput, _ time.Time) error {
	if f.unknown {
		return ErrUnknownInterest
	}
	f.saved = &in
	return nil
}
func (f *fakeRepo) UpsertPreferences(_ context.Context, _ string, p Preferences) error {
	f.prefs = &p
	return nil
}

func svcAt(now time.Time, r *fakeRepo) *Service {
	s := NewService(r)
	s.now = func() time.Time { return now }
	return s
}

func isValidation(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := map[string]int{
		"2008-06-15": 18, // birthday today
		"2008-06-16": 17, // birthday tomorrow
		"2000-01-01": 26,
		"1990-12-31": 35,
		"2024-02-29": 2, // leap-day birth
	}
	for birth, want := range cases {
		if got := AgeOn(birth, now); got != want {
			t.Errorf("AgeOn(%s) = %d, want %d", birth, got, want)
		}
	}
	if AgeOn("garbage", now) != 0 {
		t.Error("garbage date should yield 0")
	}
}

func TestSaveProfileAgeGate(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	s := svcAt(now, &fakeRepo{})
	in := ProfileInput{FirstName: " Léa ", BirthDate: "2008-06-15", Gender: "woman"}
	if err := s.SaveProfile(context.Background(), "u", in); err != nil {
		t.Fatalf("exactly 18 today must be accepted: %v", err)
	}
	in.BirthDate = "2008-06-16"
	if err := s.SaveProfile(context.Background(), "u", in); !isValidation(err) {
		t.Fatalf("17y364d must be rejected, got %v", err)
	}
}

func TestSaveProfileNormalises(t *testing.T) {
	r := &fakeRepo{}
	s := svcAt(time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), r)
	err := s.SaveProfile(context.Background(), "u", ProfileInput{
		FirstName: "  Léa  ", BirthDate: "1995-01-01", Gender: "woman", Bio: "  hello ",
		Interests: []string{"music", " music ", "", "art"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.saved.FirstName != "Léa" || r.saved.Bio != "hello" || len(r.saved.Interests) != 2 {
		t.Fatalf("not normalised: %+v", r.saved)
	}
	r.unknown = true
	if err := s.SaveProfile(context.Background(), "u", ProfileInput{FirstName: "A", BirthDate: "1995-01-01", Gender: "man"}); !isValidation(err) {
		t.Fatalf("unknown interest should be a validation error, got %v", err)
	}
}

func TestSavePreferences(t *testing.T) {
	s := svcAt(time.Now(), &fakeRepo{})
	ok := Preferences{InterestedIn: []string{"man", "man"}, MinAge: 18, MaxAge: 99}
	if err := s.SavePreferences(context.Background(), "u", ok); err != nil {
		t.Fatal(err)
	}
	d := 0
	for _, bad := range []Preferences{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 30, MaxAge: 20},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 100},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: &d},
	} {
		if err := s.SavePreferences(context.Background(), "u", bad); !isValidation(err) {
			t.Errorf("expected validation error for %+v, got %v", bad, err)
		}
	}
}

func TestCoarse(t *testing.T) {
	if got := Coarse(45.764043); got != 45.76 {
		t.Fatalf("got %v", got)
	}
	if got := Coarse(-4.835659); got != -4.84 {
		t.Fatalf("got %v", got)
	}
}
