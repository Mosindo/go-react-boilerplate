package profiles

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeRepo struct {
	Repository
	saved    *StoredProfile
	existing *StoredProfile
}

func (f *fakeRepo) ListInterests(context.Context) ([]Interest, error) {
	return []Interest{{Slug: "music", Label: "Musique"}, {Slug: "travel", Label: "Voyages"}}, nil
}

func (f *fakeRepo) GetProfile(context.Context, string) (StoredProfile, error) {
	if f.existing == nil {
		return StoredProfile{}, ErrRepositoryNotFound
	}
	return *f.existing, nil
}

func (f *fakeRepo) UpsertProfile(_ context.Context, p StoredProfile) error {
	f.saved = &p
	return nil
}

func newTestService(repo Repository, now time.Time) *Service {
	s := NewService(repo, func(id string) string { return "/media/" + id })
	s.now = func() time.Time { return now }
	return s
}

func TestAgeOn(t *testing.T) {
	today := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
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
		if got := AgeOn(b, today); got != c.want {
			t.Errorf("AgeOn(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestApproxDistance(t *testing.T) {
	cases := map[float64]int{0: 1, 0.4: 1, 1.2: 2, 9.9: 10, 10: 10, 10.1: 15, 23: 25, 99.9: 100}
	for in, want := range cases {
		if got := ApproxDistance(in); got != want {
			t.Errorf("ApproxDistance(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestRoundCoordinate(t *testing.T) {
	if got := RoundCoordinate(48.856789); got != 48.86 {
		t.Errorf("got %v", got)
	}
	if got := RoundCoordinate(-2.3449); got != -2.34 {
		t.Errorf("got %v", got)
	}
}

func TestSaveProfileEnforcesAgeGate(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{}
	svc := newTestService(repo, now)
	in := ProfileInput{FirstName: "Léa", BirthDate: "2008-06-16", Gender: "woman", Latitude: 48.85, Longitude: 2.35}

	if err := svc.SaveProfile(context.Background(), "u1", in); !IsValidation(err) {
		t.Fatalf("a 17-year-old must be rejected, got %v", err)
	}
	in.BirthDate = "2008-06-15"
	if err := svc.SaveProfile(context.Background(), "u1", in); err != nil {
		t.Fatalf("exactly 18 must be accepted: %v", err)
	}
	if repo.saved == nil || repo.saved.LookingFor != "open" {
		t.Fatalf("defaults not applied: %+v", repo.saved)
	}
}

func TestBirthDateIsImmutable(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	birth := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepo{existing: &StoredProfile{BirthDate: birth}}
	svc := newTestService(repo, now)

	in := ProfileInput{FirstName: "Léa", BirthDate: "1985-01-01", Gender: "woman", Latitude: 1, Longitude: 1}
	if err := svc.SaveProfile(context.Background(), "u1", in); err != ErrBirthDateLocked {
		t.Fatalf("expected ErrBirthDateLocked, got %v", err)
	}
	in.BirthDate = "1990-01-01"
	if err := svc.SaveProfile(context.Background(), "u1", in); err != nil {
		t.Fatalf("same birth date must be accepted: %v", err)
	}
}

func TestInterestsAreValidatedAndDeduplicated(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{}
	svc := newTestService(repo, now)
	in := ProfileInput{FirstName: "Léa", BirthDate: "1990-01-01", Gender: "woman", Interests: []string{"music", "music", "travel"}}
	if err := svc.SaveProfile(context.Background(), "u1", in); err != nil {
		t.Fatal(err)
	}
	if len(repo.saved.Interests) != 2 {
		t.Fatalf("duplicates must collapse: %v", repo.saved.Interests)
	}
	in.Interests = []string{"quantum-chess"}
	if err := svc.SaveProfile(context.Background(), "u1", in); !IsValidation(err) {
		t.Fatalf("unknown interest must be rejected, got %v", err)
	}
}

func TestCleanText(t *testing.T) {
	got, err := cleanText("  Hello\x00 ‮\tWorld\r\n! ", "name", 1, 40, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(got, "\x00\t\r\n") {
		t.Errorf("control characters survived: %q", got)
	}
	if _, err := cleanText("   ", "name", 1, 40, false); !IsValidation(err) {
		t.Errorf("blank value must fail: %v", err)
	}
	if _, err := cleanText(strings.Repeat("é", 41), "name", 1, 40, false); !IsValidation(err) {
		t.Errorf("limit is counted in characters, not bytes: %v", err)
	}
	if _, err := cleanText(strings.Repeat("é", 40), "name", 1, 40, false); err != nil {
		t.Errorf("40 multibyte characters are fine: %v", err)
	}
	bio, _ := cleanText("line1\nline2", "bio", 0, 500, true)
	if bio != "line1\nline2" {
		t.Errorf("bio keeps newlines: %q", bio)
	}
}
