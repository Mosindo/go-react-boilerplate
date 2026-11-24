package profiles

import (
	"errors"
	"testing"
	"time"
)

var now = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func validRequest() UpsertProfileRequest {
	return UpsertProfileRequest{FirstName: "  Camille  ", Birthdate: "1995-03-10", Gender: "woman", Bio: " Hello\x00 ", City: "Lyon"}
}

func TestAgeAt(t *testing.T) {
	birth := time.Date(2008, 6, 16, 0, 0, 0, 0, time.UTC)
	if AgeAt(birth, now) != 17 {
		t.Fatal("birthday tomorrow: still 17")
	}
	if AgeAt(birth.AddDate(0, 0, -1), now) != 18 {
		t.Fatal("birthday today: 18")
	}
}

func TestNormalizeProfile(t *testing.T) {
	p, err := NormalizeProfile(validRequest(), now)
	if err != nil {
		t.Fatal(err)
	}
	if p.FirstName != "Camille" || p.Bio != "Hello" || p.RelationshipGoal != "unsure" {
		t.Fatalf("unexpected normalisation %+v", p)
	}

	cases := []struct {
		mutate func(*UpsertProfileRequest)
		want   error
	}{
		{func(r *UpsertProfileRequest) { r.FirstName = "<script>" }, ErrInvalidFirstName},
		{func(r *UpsertProfileRequest) { r.Birthdate = "2010-01-01" }, ErrUnderage},
		{func(r *UpsertProfileRequest) { r.Birthdate = "2030-01-01" }, ErrInvalidBirthdate},
		{func(r *UpsertProfileRequest) { r.Birthdate = "1990-02-30" }, ErrInvalidBirthdate},
		{func(r *UpsertProfileRequest) { r.Gender = "robot" }, ErrInvalidGender},
		{func(r *UpsertProfileRequest) { r.RelationshipGoal = "marriage-now" }, ErrInvalidGoal},
		{func(r *UpsertProfileRequest) { h := 300; r.HeightCm = &h }, ErrInvalidHeight},
	}
	for i, tc := range cases {
		req := validRequest()
		tc.mutate(&req)
		if _, err := NormalizeProfile(req, now); !errors.Is(err, tc.want) {
			t.Errorf("case %d: expected %v, got %v", i, tc.want, err)
		}
	}
}

func TestLocationRoundingAndDistance(t *testing.T) {
	lat, lng, err := NormalizeLocation(45.764043, 4.835659)
	if err != nil || lat != 45.76 || lng != 4.84 {
		t.Fatalf("unexpected rounding %v %v %v", lat, lng, err)
	}
	if _, _, err := NormalizeLocation(91, 0); !errors.Is(err, ErrInvalidLocation) {
		t.Fatal("expected out of range latitude to fail")
	}
	if ApproximateDistance(0.2) != 1 || ApproximateDistance(4.4) != 4 || ApproximateDistance(23) != 25 {
		t.Fatal("unexpected distance approximation")
	}
}

func TestNormalizePreferences(t *testing.T) {
	p, err := NormalizePreferences(UpdatePreferencesRequest{InterestedIn: []string{"man", "man"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(p.InterestedIn) != 1 {
		t.Fatalf("unexpected %v %v", p, err)
	}
	if _, err := NormalizePreferences(UpdatePreferencesRequest{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30}); !errors.Is(err, ErrInvalidAgeRange) {
		t.Fatal("expected minors to be rejected from age range")
	}
	if _, err := NormalizePreferences(UpdatePreferencesRequest{InterestedIn: []string{"x"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30}); !errors.Is(err, ErrInvalidInterested) {
		t.Fatal("expected unknown gender rejection")
	}
}

func TestNormalizeInterestIDs(t *testing.T) {
	if ids, err := NormalizeInterestIDs([]int{1, 1, 2}); err != nil || len(ids) != 2 {
		t.Fatalf("expected dedupe, got %v %v", ids, err)
	}
	if _, err := NormalizeInterestIDs([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}); !errors.Is(err, ErrTooManyInterests) {
		t.Fatal("expected too many interests")
	}
}
