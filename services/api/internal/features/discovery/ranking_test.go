package discovery

import (
	"testing"
	"time"
)

func ptr(f float64) *float64 { return &f }

func TestScoreRankerOrdersBySignals(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	viewer := Viewer{MaxDistanceKm: 50}
	old := now.Add(-90 * 24 * time.Hour)
	candidates := []Candidate{
		{UserID: "inactive-far", DistanceKm: ptr(49), LastActiveAt: now.Add(-30 * 24 * time.Hour), CreatedAt: old, PhotoCount: 1},
		{UserID: "active-close", DistanceKm: ptr(2), LastActiveAt: now.Add(-time.Hour), CreatedAt: old, PhotoCount: 3, HasBio: true, SharedInterests: 2},
		{UserID: "liked-me", DistanceKm: ptr(30), LastActiveAt: now.Add(-48 * time.Hour), CreatedAt: old, PhotoCount: 1, LikedViewer: true},
	}
	ranked := NewScoreRanker().Rank(viewer, candidates, now)
	if ranked[0].UserID != "liked-me" || ranked[1].UserID != "active-close" || ranked[2].UserID != "inactive-far" {
		t.Fatalf("unexpected order: %s, %s, %s", ranked[0].UserID, ranked[1].UserID, ranked[2].UserID)
	}
}

func TestScoreSharedInterestsCapped(t *testing.T) {
	now := time.Now()
	r := NewScoreRanker()
	base := Candidate{LastActiveAt: now, CreatedAt: now.Add(-365 * 24 * time.Hour)}
	five, twenty := base, base
	five.SharedInterests = 5
	twenty.SharedInterests = 20
	if r.Score(Viewer{}, five, now) != r.Score(Viewer{}, twenty, now) {
		t.Fatal("shared interests contribution must be capped")
	}
}

func TestRankIsDeterministicOnTies(t *testing.T) {
	now := time.Now()
	c := Candidate{LastActiveAt: now, CreatedAt: now}
	a, b := c, c
	a.UserID, b.UserID = "b", "a"
	ranked := NewScoreRanker().Rank(Viewer{}, []Candidate{a, b}, now)
	if ranked[0].UserID != "a" {
		t.Fatal("ties must be broken by user id")
	}
}
