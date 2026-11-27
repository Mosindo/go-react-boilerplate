package matching

import (
	"testing"
	"time"
)

func TestScoreRankerPrefersSharedInterestsAndActivity(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	near := 3.0
	far := 80.0
	candidates := []Candidate{
		{UserID: "dormant", LastActiveAt: now.Add(-60 * 24 * time.Hour), SharedInterests: 1, DistanceKm: &far},
		{UserID: "shared", LastActiveAt: now.Add(-2 * time.Hour), SharedInterests: 3, DistanceKm: &near},
		{UserID: "active", LastActiveAt: now.Add(-1 * time.Hour), SharedInterests: 0, DistanceKm: &far},
	}
	ranked := ScoreRanker{}.Rank("viewer", candidates, now)
	if ranked[0].UserID != "shared" || ranked[2].UserID != "dormant" {
		t.Fatalf("unexpected ranking: %v, %v, %v", ranked[0].UserID, ranked[1].UserID, ranked[2].UserID)
	}
	if candidates[0].UserID != "dormant" {
		t.Fatal("Rank must not mutate its input")
	}
}

func TestScoreRankerIsDeterministicWithinADay(t *testing.T) {
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	c := []Candidate{{UserID: "a", LastActiveAt: now}, {UserID: "b", LastActiveAt: now}, {UserID: "c", LastActiveAt: now}}
	first := ScoreRanker{}.Rank("viewer", c, now)
	second := ScoreRanker{}.Rank("viewer", c, now.Add(3*time.Hour))
	for i := range first {
		if first[i].UserID != second[i].UserID {
			t.Fatal("ranking should be stable for a viewer within the same day")
		}
	}
}

func TestLikedViewerBoost(t *testing.T) {
	now := time.Now()
	base := Candidate{UserID: "x", LastActiveAt: now}
	liked := base
	liked.LikedViewer = true
	r := ScoreRanker{}
	if r.Score("v", liked, now) <= r.Score("v", base, now) {
		t.Fatal("a candidate who liked the viewer should score higher")
	}
}
