package matching

import (
	"testing"
	"time"
)

func ptr(v float64) *float64 { return &v }

func TestDefaultRankerPrefersActiveCloseAndShared(t *testing.T) {
	now := time.Now()
	candidates := []Candidate{
		{UserID: "stale", LastActiveAt: now.Add(-30 * 24 * time.Hour), DistanceKm: ptr(2), SharedInterests: 0},
		{UserID: "best", LastActiveAt: now, DistanceKm: ptr(1), SharedInterests: 4},
		{UserID: "far", LastActiveAt: now, DistanceKm: ptr(200), SharedInterests: 0},
	}
	ranked := DefaultRanker{}.Rank(candidates, now)
	if ranked[0].UserID != "best" || ranked[2].UserID != "stale" {
		t.Fatalf("unexpected order: %v, %v, %v", ranked[0].UserID, ranked[1].UserID, ranked[2].UserID)
	}
	if candidates[0].UserID != "stale" {
		t.Fatal("Rank must not mutate its input")
	}
}

func TestScoreIsBounded(t *testing.T) {
	r := DefaultRanker{}
	s := r.Score(Candidate{LastActiveAt: time.Now(), DistanceKm: ptr(0), SharedInterests: 99, LikedViewer: true}, time.Now())
	if s > 1.0001 || s < 0 {
		t.Fatalf("score out of bounds: %v", s)
	}
}
