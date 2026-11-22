package matching

import (
	"testing"
	"time"
)

func TestWeightedRankerPrefersCloseActiveCompatibleProfiles(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	pool := []Candidate{
		{UserID: "far-inactive", DistanceKm: 45, LastActiveAt: now.AddDate(0, 0, -30)},
		{UserID: "close-active", DistanceKm: 2, LastActiveAt: now.Add(-time.Hour), SharedInterests: 3},
		{UserID: "mid", DistanceKm: 20, LastActiveAt: now.AddDate(0, 0, -2), SharedInterests: 1},
	}
	ranked := DefaultRanker().Rank(now, 50, pool)
	if ranked[0].UserID != "close-active" || ranked[2].UserID != "far-inactive" {
		t.Fatalf("unexpected order: %v, %v, %v", ranked[0].UserID, ranked[1].UserID, ranked[2].UserID)
	}
	if pool[0].UserID != "far-inactive" {
		t.Fatalf("Rank must not mutate its input")
	}
}

func TestWeightedRankerReciprocalBoost(t *testing.T) {
	now := time.Now()
	a := Candidate{UserID: "a", DistanceKm: 10, LastActiveAt: now}
	b := a
	b.UserID = "b"
	b.LikedViewer = true
	r := DefaultRanker()
	if r.Score(now, 50, b) <= r.Score(now, 50, a) {
		t.Fatalf("a candidate who liked the viewer should rank higher, all else equal")
	}
}

func TestPairKeyIsOrderIndependent(t *testing.T) {
	a, b := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	if PairKey(a, b) != PairKey(b, a) {
		t.Fatalf("pair key must be symmetric")
	}
	lo, hi := OrderedPair(b, a)
	if lo != a || hi != b {
		t.Fatalf("unexpected ordering %s %s", lo, hi)
	}
}
