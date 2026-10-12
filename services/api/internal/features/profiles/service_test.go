package profiles

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today: already 18
		{"2008-06-16", 17}, // birthday tomorrow: still 17
		{"2000-12-31", 25},
		{"1990-01-01", 36},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := ageOn(b, now); got != c.want {
			t.Errorf("ageOn(%s)=%d want %d", c.birth, got, c.want)
		}
	}
}

func TestDistanceBucketNeverExact(t *testing.T) {
	cases := map[float64]int{0: 1, 0.4: 1, 1.5: 2, 4.9: 5, 7: 10, 24: 25, 26: 50, 99: 100, 400: 500, 9000: 1000}
	for km, want := range cases {
		if got := DistanceBucket(km); got != want {
			t.Errorf("DistanceBucket(%v)=%d want %d", km, got, want)
		}
	}
}

func TestRoundCoordinate(t *testing.T) {
	if got := RoundCoordinate(48.856614); got != 48.86 {
		t.Fatalf("got %v", got)
	}
	if got := RoundCoordinate(-122.4194); got != -122.42 {
		t.Fatalf("got %v", got)
	}
}

// fakeRepo implements only what UpdateProfile needs.
type fakeRepo struct {
	Repository
	existing *StoredProfile
	saved    *ProfileInput
}

func (f *fakeRepo) GetOwn(context.Context, string) (StoredProfile, error) {
	if f.existing == nil {
		return StoredProfile{}, ErrRepositoryNotFound
	}
	return *f.existing, nil
}
func (f *fakeRepo) UpsertProfile(_ context.Context, _ string, in ProfileInput, _ []string) error {
	f.saved = &in
	return nil
}

func newTestService(repo Repository) *Service {
	s := NewService(repo, nil)
	s.now = func() time.Time { return time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) }
	return s
}

func TestUpdateProfileValidation(t *testing.T) {
	valid := UpdateProfileRequest{FirstName: "  Zoë-Marie ", BirthDate: "1995-05-05", Gender: "woman", Bio: "Hi\nthere"}
	cases := map[string]func(r *UpdateProfileRequest){
		"underage":        func(r *UpdateProfileRequest) { r.BirthDate = "2008-06-16" },
		"too old":         func(r *UpdateProfileRequest) { r.BirthDate = "1900-01-01" },
		"future":          func(r *UpdateProfileRequest) { r.BirthDate = "2030-01-01" },
		"bad date":        func(r *UpdateProfileRequest) { r.BirthDate = "yesterday" },
		"empty name":      func(r *UpdateProfileRequest) { r.FirstName = "   " },
		"digits in name":  func(r *UpdateProfileRequest) { r.FirstName = "R2D2" },
		"long name":       func(r *UpdateProfileRequest) { r.FirstName = strings.Repeat("a", 41) },
		"bad gender":      func(r *UpdateProfileRequest) { r.Gender = "x" },
		"long bio":        func(r *UpdateProfileRequest) { r.Bio = strings.Repeat("é", 501) },
		"control in bio":  func(r *UpdateProfileRequest) { r.Bio = "a\x00b" },
		"too many tags":   func(r *UpdateProfileRequest) { r.Interests = strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",") },
		"control in city": func(r *UpdateProfileRequest) { r.City = "Pa\tris" },
	}
	for name, mutate := range cases {
		repo := &fakeRepo{}
		req := valid
		mutate(&req)
		err := newTestService(repo).UpdateProfile(context.Background(), "u1", req)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: expected validation error, got %v", name, err)
		}
		if repo.saved != nil {
			t.Errorf("%s: invalid input must not be saved", name)
		}
	}

	repo := &fakeRepo{}
	if err := newTestService(repo).UpdateProfile(context.Background(), "u1", valid); err != nil {
		t.Fatal(err)
	}
	if repo.saved == nil || repo.saved.FirstName != "Zoë-Marie" {
		t.Fatalf("name should be trimmed and saved, got %+v", repo.saved)
	}
}

func TestBirthDateIsLocked(t *testing.T) {
	birth, _ := time.Parse("2006-01-02", "1995-05-05")
	repo := &fakeRepo{existing: &StoredProfile{BirthDate: birth}}
	req := UpdateProfileRequest{FirstName: "Ana", BirthDate: "1990-01-01", Gender: "woman"}
	if err := newTestService(repo).UpdateProfile(context.Background(), "u1", req); !errors.Is(err, ErrBirthDateLocked) {
		t.Fatalf("expected ErrBirthDateLocked, got %v", err)
	}
	req.BirthDate = "1995-05-05"
	if err := newTestService(repo).UpdateProfile(context.Background(), "u1", req); err != nil {
		t.Fatalf("same birth date must be accepted: %v", err)
	}
}

func TestUpdatePreferencesValidation(t *testing.T) {
	svc := newTestService(&fakeRepo{})
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 17, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 100, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 40, AgeMax: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, AgeMin: 18, AgeMax: 30, MaxDistanceKm: 99999},
	}
	for i, req := range bad {
		var ve *ValidationError
		if err := svc.UpdatePreferences(context.Background(), "u", req); !errors.As(err, &ve) {
			t.Errorf("case %d: expected validation error, got %v", i, err)
		}
	}
}
