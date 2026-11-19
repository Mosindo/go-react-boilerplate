package matching

import (
	"testing"
	"time"
)

func TestDefaultRankerOrdersBySignals(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	near, far := 2.0, 45.0
	viewer := Viewer{MaxDistanceKm: 50}
	candidates := []Candidate{
		{UserID: "inactive", LastActiveAt: now.AddDate(0, 0, -60), DistanceKm: &near},
		{UserID: "far", LastActiveAt: now, DistanceKm: &far},
		{UserID: "near", LastActiveAt: now, DistanceKm: &near},
		{UserID: "liked-me", LastActiveAt: now.AddDate(0, 0, -5), DistanceKm: &far, LikedViewer: true},
		{UserID: "shared", LastActiveAt: now, DistanceKm: &far, SharedInterests: 4},
	}
	got := DefaultRanker{}.Rank(viewer, candidates, now)
	order := []string{"liked-me", "shared", "near", "far", "inactive"}
	for i, id := range order {
		if got[i].UserID != id {
			t.Fatalf("position %d: got %s, want %s (full: %+v)", i, got[i].UserID, id, got)
		}
	}
}

func TestScoreIsDeterministicForTies(t *testing.T) {
	now := time.Now()
	c := []Candidate{{UserID: "b", LastActiveAt: now}, {UserID: "a", LastActiveAt: now}}
	got := DefaultRanker{}.Rank(Viewer{}, c, now)
	if got[0].UserID != "a" {
		t.Fatal("ties must be broken by id for stable pagination")
	}
}
