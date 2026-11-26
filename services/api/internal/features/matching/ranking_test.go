package matching

import (
	"testing"
	"time"
)

func ptr(f float64) *float64 { return &f }

func TestScoreFavoursRelevantProfiles(t *testing.T) {
	now := time.Now()
	base := Candidate{LastActiveAt: now.Add(-60 * 24 * time.Hour), DistanceKm: ptr(45)}

	active := base
	active.LastActiveAt = now.Add(-time.Hour)
	if Score(active, 50, now) <= Score(base, 50, now) {
		t.Fatal("recent activity should increase the score")
	}

	close := base
	close.DistanceKm = ptr(2)
	if Score(close, 50, now) <= Score(base, 50, now) {
		t.Fatal("proximity should increase the score")
	}

	shared := base
	shared.SharedInterests = 3
	if Score(shared, 50, now) <= Score(base, 50, now) {
		t.Fatal("shared interests should increase the score")
	}

	capped := base
	capped.SharedInterests = 50
	five := base
	five.SharedInterests = 5
	if Score(capped, 50, now) != Score(five, 50, now) {
		t.Fatal("shared interests bonus must be capped")
	}
}

func TestRankIsDeterministicWithoutJitter(t *testing.T) {
	now := time.Now()
	r := &ScoreRanker{}
	out := r.Rank([]Candidate{
		{UserID: "far", LastActiveAt: now, DistanceKm: ptr(49)},
		{UserID: "near", LastActiveAt: now, DistanceKm: ptr(1)},
		{UserID: "liked", LastActiveAt: now, DistanceKm: ptr(49), LikedMe: true},
	}, 50, now)
	if out[0].UserID != "liked" || out[1].UserID != "near" || out[2].UserID != "far" {
		t.Fatalf("unexpected order: %v, %v, %v", out[0].UserID, out[1].UserID, out[2].UserID)
	}
}

func TestBoundingBox(t *testing.T) {
	box := boundingBox(48.85, 2.35, 50)
	if box == nil {
		t.Fatal("expected a box")
	}
	if box.latMin >= 48.85 || box.latMax <= 48.85 || box.lngMin >= 2.35 || box.lngMax <= 2.35 {
		t.Fatalf("box does not contain origin: %+v", box)
	}
	if boundingBox(89.9, 0, 50) != nil {
		t.Fatal("no longitude box near the poles")
	}
}

func TestPairKeyIsSymmetric(t *testing.T) {
	a, b := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	if pairKey(a, b) != pairKey(b, a) {
		t.Fatal("pair key must not depend on order")
	}
}
