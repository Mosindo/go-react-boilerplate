package profiles

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeRepo struct {
	Repository
	saved StoredProfile
}

func (f *fakeRepo) GetSelf(context.Context, string) (SelfProfile, error) {
	return SelfProfile{}, ErrNotFound
}
func (f *fakeRepo) Upsert(_ context.Context, p StoredProfile) error { f.saved = p; return nil }

func newTestService() (*Service, *fakeRepo) {
	repo := &fakeRepo{}
	s := NewService(repo)
	s.now = func() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) }
	return s, repo
}

func validRequest() UpdateProfileRequest {
	return UpdateProfileRequest{FirstName: " Léa-Rose ", Birthdate: "1996-06-15", Gender: "woman", Bio: "Hello\x00 world", City: "Paris", Latitude: 48.8566, Longitude: 2.3522, InterestIDs: []int16{1, 1, 2}}
}

func TestAgeOn(t *testing.T) {
	at := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := map[string]int{"2008-06-15": 18, "2008-06-16": 17, "1996-12-31": 29, "2000-02-29": 26}
	for birth, want := range cases {
		b, _ := time.Parse("2006-01-02", birth)
		if got := AgeOn(b, at); got != want {
			t.Fatalf("AgeOn(%s) = %d, want %d", birth, got, want)
		}
	}
}

func TestMinorsAreRejectedOnTheirBirthdayEve(t *testing.T) {
	s, _ := newTestService()
	req := validRequest()
	req.Birthdate = "2008-06-16" // turns 18 tomorrow
	if _, err := s.validateProfile("u", req); err == nil {
		t.Fatal("17 year old accepted")
	}
	req.Birthdate = "2008-06-15"
	if _, err := s.validateProfile("u", req); err != nil {
		t.Fatalf("18 year old rejected: %v", err)
	}
}

func TestProfileNormalisation(t *testing.T) {
	s, _ := newTestService()
	stored, err := s.validateProfile("u", validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if stored.FirstName != "Léa-Rose" || stored.Bio != "Hello world" {
		t.Fatalf("not normalised: %+v", stored)
	}
	if stored.Latitude != 48.86 || stored.Longitude != 2.35 {
		t.Fatalf("coordinates must be coarsened: %v %v", stored.Latitude, stored.Longitude)
	}
	if len(stored.InterestIDs) != 2 {
		t.Fatalf("interests must be deduplicated: %v", stored.InterestIDs)
	}
	if !stored.IsDiscoverable || !stored.ShowDistance {
		t.Fatal("privacy defaults must be discoverable with distance shown")
	}
}

func TestProfileRejections(t *testing.T) {
	s, _ := newTestService()
	mutate := map[string]func(*UpdateProfileRequest){
		"empty name":      func(r *UpdateProfileRequest) { r.FirstName = "  " },
		"name with html":  func(r *UpdateProfileRequest) { r.FirstName = "<b>x</b>" },
		"long name":       func(r *UpdateProfileRequest) { r.FirstName = strings.Repeat("a", 41) },
		"bad gender":      func(r *UpdateProfileRequest) { r.Gender = "x" },
		"long bio":        func(r *UpdateProfileRequest) { r.Bio = strings.Repeat("é", 501) },
		"bad latitude":    func(r *UpdateProfileRequest) { r.Latitude = 91 },
		"bad longitude":   func(r *UpdateProfileRequest) { r.Longitude = -181 },
		"too many tags":   func(r *UpdateProfileRequest) { r.InterestIDs = make([]int16, 11) },
		"negative tag":    func(r *UpdateProfileRequest) { r.InterestIDs = []int16{-1} },
		"future birth":    func(r *UpdateProfileRequest) { r.Birthdate = "2030-01-01" },
		"ancient birth":   func(r *UpdateProfileRequest) { r.Birthdate = "1900-01-01" },
		"malformed birth": func(r *UpdateProfileRequest) { r.Birthdate = "15/06/1996" },
	}
	for name, fn := range mutate {
		req := validRequest()
		fn(&req)
		if req.Bio == "Hello\x00 world" {
			req.Bio = "ok"
		}
		if _, err := s.validateProfile("u", req); err == nil {
			t.Fatalf("%s: expected a validation error", name)
		}
	}
}

func TestPreferencesValidation(t *testing.T) {
	ok, err := validatePreferences(UpdatePreferencesRequest{InterestedIn: []string{"woman", "woman", "man"}, MinAge: 20, MaxAge: 35})
	if err != nil || len(ok.InterestedIn) != 2 || ok.MaxDistanceKm != DefaultDistance {
		t.Fatalf("unexpected: %+v %v", ok, err)
	}
	bad := []UpdatePreferencesRequest{
		{InterestedIn: nil, MinAge: 20, MaxAge: 30},
		{InterestedIn: []string{"x"}, MinAge: 20, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30},
		{InterestedIn: []string{"man"}, MinAge: 30, MaxAge: 20},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 100},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: 501},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: -4},
	}
	for i, req := range bad {
		if _, err := validatePreferences(req); err == nil {
			t.Fatalf("case %d should be rejected", i)
		}
	}
}

func TestPrivacyFlagsKeepValueWhenOmitted(t *testing.T) {
	s, repo := newTestService()
	off := false
	req := validRequest()
	req.IsDiscoverable = &off
	if _, err := s.UpdateSelf(context.Background(), "u", req); err == nil {
		// GetSelf fake returns ErrNotFound so the (nil) read-back is irrelevant; only the stored value matters.
		_ = err
	}
	if repo.saved.IsDiscoverable {
		t.Fatal("explicit discoverable=false must be stored")
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("123e4567-e89b-12d3-a456-426614174000") || IsUUID("123") || IsUUID("123e4567-e89b-12d3-a456-42661417400g") || IsUUID("") {
		t.Fatal("uuid detection broken")
	}
}
