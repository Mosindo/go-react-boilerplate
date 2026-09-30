package discovery

import (
	"testing"
	"time"
)

func TestRankPrefersReciprocityInterestsAndActivity(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	near, far := 3.0, 80.0
	old := now.Add(-60 * 24 * time.Hour)
	candidates := []Candidate{
		{UserID: "a-dormant", LastActiveAt: now.Add(-40 * 24 * time.Hour), CreatedAt: old, DistanceKm: &near},
		{UserID: "b-liked-me", LikedViewer: true, LastActiveAt: now.Add(-2 * time.Hour), CreatedAt: old, DistanceKm: &far},
		{UserID: "c-shared", SharedInterests: 4, LastActiveAt: now.Add(-2 * time.Hour), CreatedAt: old, DistanceKm: &far},
	}
	ranked := Rank(candidates, DefaultScorer{}, now)
	if ranked[0].UserID != "b-liked-me" || ranked[1].UserID != "c-shared" || ranked[2].UserID != "a-dormant" {
		t.Fatalf("unexpected ranking: %s, %s, %s", ranked[0].UserID, ranked[1].UserID, ranked[2].UserID)
	}
}

func TestRankIsDeterministicOnTies(t *testing.T) {
	now := time.Now()
	c := []Candidate{{UserID: "b", LastActiveAt: now, CreatedAt: now}, {UserID: "a", LastActiveAt: now, CreatedAt: now}}
	if ranked := Rank(c, DefaultScorer{}, now); ranked[0].UserID != "a" {
		t.Fatal("ties must be broken by user id")
	}
}
