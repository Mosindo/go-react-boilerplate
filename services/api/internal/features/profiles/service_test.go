package profiles

import (
	"context"
	"strings"
	"testing"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
)

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"2000-02-29", 26},
		{"1990-12-31", 35},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeAt(b, now); got != c.want {
			t.Errorf("AgeAt(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestApproxDistance(t *testing.T) {
	if ApproxDistance(nil) != nil {
		t.Fatal("nil in, nil out")
	}
	for in, want := range map[float64]int{0: 5, 0.2: 5, 5: 5, 5.01: 10, 12.4: 15, 99: 100} {
		in := in
		if got := *ApproxDistance(&in); got != want {
			t.Errorf("ApproxDistance(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestTextCleaning(t *testing.T) {
	if got := cleanText("  Jo \t\n  Ann\x00e "); got != "Jo Anne" {
		t.Errorf("cleanText = %q", got)
	}
	if got := cleanMultiline("a\r\n\r\n\r\n\r\nb\x07"); got != "a\n\nb" {
		t.Errorf("cleanMultiline = %q", got)
	}
	if _, err := normalizeInterests([]string{"a", "", "b"}); err == nil {
		t.Error("blank interest accepted")
	}
	got, err := normalizeInterests([]string{"Yoga", "yoga ", "travel"})
	if err != nil || len(got) != 2 {
		t.Errorf("dedupe: %v %v", got, err)
	}
}

type fakeRepo struct {
	Repository
	saved *ProfileInput
	own   *OwnProfile
}

func (f *fakeRepo) GetOwn(context.Context, string) (*OwnProfile, error) {
	if f.own == nil {
		return nil, ErrNotFound
	}
	return f.own, nil
}

func (f *fakeRepo) Save(_ context.Context, in ProfileInput) error {
	f.saved = &in
	f.own = &OwnProfile{UserID: in.UserID, BirthDate: in.BirthDate.Format("2006-01-02"), ShowDistance: in.ShowDistance, Discoverable: in.Discoverable}
	return nil
}

func TestSaveEnforcesAgeGateAndImmutableBirthDate(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)
	svc.now = func() time.Time { return time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) }
	req := SaveProfileRequest{DisplayName: "Sam", BirthDate: "2008-06-16", Gender: "man"}

	_, err := svc.Save(context.Background(), "u1", req)
	appErr, ok := platformerrors.As(err)
	if !ok || appErr.Fields["birthDate"] == "" {
		t.Fatalf("17-year-old accepted: %v", err)
	}
	if repo.saved != nil {
		t.Fatal("underage profile persisted")
	}

	req.BirthDate = "2008-06-15"
	if _, err := svc.Save(context.Background(), "u1", req); err != nil {
		t.Fatalf("18th birthday today must pass: %v", err)
	}
	req.BirthDate = "1990-01-01"
	if _, err := svc.Save(context.Background(), "u1", req); err == nil {
		t.Fatal("birth date changed after creation")
	}

	req = SaveProfileRequest{DisplayName: "S", BirthDate: "2008-06-15", Gender: "x", Bio: strings.Repeat("é", 501)}
	_, err = svc.Save(context.Background(), "u2", req)
	appErr, _ = platformerrors.As(err)
	if appErr == nil || appErr.Fields["displayName"] == "" || appErr.Fields["gender"] == "" || appErr.Fields["bio"] == "" {
		t.Fatalf("expected three field errors, got %v", err)
	}
}

func TestSavePreferencesValidation(t *testing.T) {
	svc := NewService(&fakeRepo{})
	bad := []Preferences{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 100, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 0},
	}
	for i, p := range bad {
		if _, err := svc.SavePreferences(context.Background(), "u", p); err == nil {
			t.Errorf("case %d accepted: %+v", i, p)
		}
	}
}
