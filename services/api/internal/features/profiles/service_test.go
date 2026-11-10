package profiles

import (
	"errors"
	"testing"
	"time"
)

func fixedService(now time.Time) *Service {
	return &Service{now: func() time.Time { return now }}
}

func TestAgeAt(t *testing.T) {
	b := time.Date(2000, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := map[time.Time]int{
		time.Date(2018, 6, 14, 0, 0, 0, 0, time.UTC): 17,
		time.Date(2018, 6, 15, 0, 0, 0, 0, time.UTC): 18,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC):  25,
	}
	for now, want := range cases {
		if got := AgeAt(b, now); got != want {
			t.Errorf("AgeAt(%v) = %d, want %d", now, got, want)
		}
	}
}

func validRequest() UpsertProfileRequest {
	return UpsertProfileRequest{FirstName: "  Léa  Marie ", Birthdate: "1995-03-02", Gender: "woman", Bio: "  Hello  ", InterestIDs: []int{1, 2, 2}}
}

func TestValidateProfile(t *testing.T) {
	s := fixedService(time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC))

	in, err := s.ValidateProfile(validRequest())
	if err != nil {
		t.Fatalf("expected valid profile: %v", err)
	}
	if in.FirstName != "Léa Marie" || in.Bio != "Hello" || len(in.InterestIDs) != 2 {
		t.Fatalf("unexpected normalization %+v", in)
	}

	cases := []struct {
		name   string
		mutate func(*UpsertProfileRequest)
		want   error
	}{
		{"underage", func(r *UpsertProfileRequest) { r.Birthdate = "2009-01-01" }, ErrUnderage},
		{"17 yesterday", func(r *UpsertProfileRequest) { r.Birthdate = "2008-11-11" }, ErrUnderage},
		{"bad date", func(r *UpsertProfileRequest) { r.Birthdate = "1995-02-30" }, ErrInvalidBirthdate},
		{"empty name", func(r *UpsertProfileRequest) { r.FirstName = "   " }, ErrInvalidFirstName},
		{"name with digits", func(r *UpsertProfileRequest) { r.FirstName = "L3a" }, ErrInvalidFirstName},
		{"html in name", func(r *UpsertProfileRequest) { r.FirstName = "<script>" }, ErrInvalidFirstName},
		{"gender", func(r *UpsertProfileRequest) { r.Gender = "robot" }, ErrInvalidGender},
		{"height", func(r *UpsertProfileRequest) { h := 300; r.HeightCm = &h }, ErrInvalidHeight},
		{"goal", func(r *UpsertProfileRequest) { g := "forever"; r.RelationshipGoal = &g }, ErrInvalidGoal},
		{"interests", func(r *UpsertProfileRequest) { r.InterestIDs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9} }, ErrTooManyInterests},
		{"bio", func(r *UpsertProfileRequest) {
			b := make([]rune, 501)
			for i := range b {
				b[i] = 'a'
			}
			r.Bio = string(b)
		}, ErrTooLong},
	}
	for _, tc := range cases {
		req := validRequest()
		tc.mutate(&req)
		if _, err := s.ValidateProfile(req); !errors.Is(err, tc.want) {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}

	exactly18 := validRequest()
	exactly18.Birthdate = "2008-11-10"
	if _, err := s.ValidateProfile(exactly18); err != nil {
		t.Fatalf("18th birthday today must be accepted: %v", err)
	}
}

func TestValidatePreferences(t *testing.T) {
	ok, err := ValidatePreferences(PreferencesRequest{InterestedIn: []string{"woman", "man", "woman"}, MinAge: 25, MaxAge: 35, MaxDistanceKm: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(ok.InterestedIn) != 2 {
		t.Fatalf("expected dedupe, got %v", ok.InterestedIn)
	}
	bad := []PreferencesRequest{
		{InterestedIn: []string{"alien"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 501},
	}
	for _, req := range bad {
		if _, err := ValidatePreferences(req); !errors.Is(err, ErrInvalidPreferences) {
			t.Errorf("expected %+v to be rejected", req)
		}
	}
}

func TestMissing(t *testing.T) {
	if got := Missing(nil, nil, 0); len(got) != 4 {
		t.Fatalf("expected all steps missing, got %v", got)
	}
	lat, lon := 48.85, 2.35
	p := &Profile{Latitude: &lat, Longitude: &lon}
	if got := Missing(p, &Preferences{}, 1); len(got) != 0 {
		t.Fatalf("expected complete, got %v", got)
	}
}
