package users

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestAgeAt(t *testing.T) {
	cases := []struct {
		birth, on time.Time
		want      int
	}{
		{date(2000, time.June, 15), date(2026, time.June, 14), 25},
		{date(2000, time.June, 15), date(2026, time.June, 15), 26},
		{date(2000, time.June, 15), date(2026, time.December, 31), 26},
		{date(2008, time.February, 29), date(2026, time.February, 28), 17},
		{date(2008, time.February, 29), date(2026, time.March, 1), 18},
	}
	for _, c := range cases {
		if got := AgeAt(c.birth, c.on); got != c.want {
			t.Errorf("AgeAt(%s, %s) = %d, want %d", c.birth.Format("2006-01-02"), c.on.Format("2006-01-02"), got, c.want)
		}
	}
}

func TestApproximateDistanceRoundsUpToFiveKm(t *testing.T) {
	cases := map[float64]int{0: 5, 0.3: 5, 5: 5, 5.01: 10, 9.9: 10, 10: 10, 12: 15, 101.2: 105}
	for km, want := range cases {
		if got := ApproximateDistanceKm(km); got != want {
			t.Errorf("ApproximateDistanceKm(%v) = %d, want %d", km, got, want)
		}
	}
}

func TestRoundCoordinateKeepsTwoDecimals(t *testing.T) {
	cases := map[float64]float64{48.85661: 48.86, 2.3522: 2.35, -73.9856: -73.99, 0.004: 0, 45.005: 45.01}
	for in, want := range cases {
		if got := RoundCoordinate(in); got != want {
			t.Errorf("RoundCoordinate(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestTextNormalization(t *testing.T) {
	if got, err := normalizeFirstName("  Jean \t  Pierre\x00 "); err != nil || got != "Jean Pierre" {
		t.Errorf("first name: %q %v", got, err)
	}
	if _, err := normalizeFirstName(" \n "); !errors.Is(err, ErrInvalidFirstName) {
		t.Errorf("blank name must fail, got %v", err)
	}
	if _, err := normalizeFirstName(strings.Repeat("é", 41)); !errors.Is(err, ErrInvalidFirstName) {
		t.Errorf("41 runes must fail, got %v", err)
	}
	if _, err := normalizeFirstName(strings.Repeat("é", 40)); err != nil {
		t.Errorf("40 multi-byte runes are allowed: %v", err)
	}

	bio, err := normalizeBio("  Line one \r\n\r\n\r\n\r\nLine   two\x07  ")
	if err != nil || bio != "Line one\n\nLine two" {
		t.Errorf("bio: %q %v", bio, err)
	}
	if _, err := normalizeBio(strings.Repeat("x", 501)); !errors.Is(err, ErrInvalidBio) {
		t.Errorf("501 chars must fail, got %v", err)
	}

	slugs, err := normalizeInterestSlugs([]string{" Cinema ", "cinema", "voyages"})
	if err != nil || len(slugs) != 2 || slugs[0] != "cinema" {
		t.Errorf("slugs: %v %v", slugs, err)
	}
	if _, err := normalizeInterestSlugs([]string{""}); err == nil {
		t.Error("blank slug must fail")
	}
}

type stubRepo struct {
	Repository
	profile *Profile
}

func (r stubRepo) GetProfile(context.Context, string) (Profile, error) {
	if r.profile == nil {
		return Profile{}, ErrRepositoryNotFound
	}
	return *r.profile, nil
}

func (r stubRepo) SaveProfile(context.Context, Profile, []string) error { return nil }

func TestSaveProfileEnforcesAgeRules(t *testing.T) {
	now := date(2026, time.October, 15)
	svc := NewService(stubRepo{}, nil)
	svc.now = func() time.Time { return now }
	base := ProfileInput{FirstName: "Sam", Gender: "man", Bio: "", Interests: nil}

	cases := []struct {
		name  string
		birth string
		want  error
	}{
		{"exactly 18 today", "2008-10-15", nil},
		{"one day short of 18", "2008-10-16", ErrUnderage},
		{"adult", "1990-01-01", nil},
		{"99 years old", "1927-06-01", nil},
		{"100 years old", "1926-10-15", ErrInvalidAge},
		{"future", "2027-01-01", ErrInvalidAge},
		{"garbage", "yesterday", ErrInvalidBirthDate},
	}
	for _, c := range cases {
		in := base
		in.BirthDate = c.birth
		err := svc.SaveProfile(context.Background(), "user-1", in)
		if !errors.Is(err, c.want) && !(c.want == nil && err == nil) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

func TestBirthDateCannotChange(t *testing.T) {
	now := date(2026, time.October, 15)
	existing := Profile{UserID: "u", BirthDate: date(1990, time.January, 1)}
	svc := NewService(stubRepo{profile: &existing}, nil)
	svc.now = func() time.Time { return now }

	in := ProfileInput{FirstName: "Sam", BirthDate: "1991-01-01", Gender: "man"}
	if err := svc.SaveProfile(context.Background(), "u", in); !errors.Is(err, ErrBirthDateLocked) {
		t.Fatalf("expected ErrBirthDateLocked, got %v", err)
	}
	in.BirthDate = "1990-01-01"
	if err := svc.SaveProfile(context.Background(), "u", in); err != nil {
		t.Fatalf("resending the same birth date is fine: %v", err)
	}
}
