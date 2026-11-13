package matching

import (
	"testing"
	"time"
)

func TestWeightedRankerPrefersActiveCloseAndCompatible(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	r := DefaultRanker()

	candidates := []Candidate{
		{UserID: "stale-far", DistanceKm: 45, MaxDistanceKm: 50, LastActiveAt: now.Add(-30 * 24 * time.Hour)},
		{UserID: "active-close-shared", DistanceKm: 2, MaxDistanceKm: 50, LastActiveAt: now.Add(-time.Hour), SharedInterests: 4},
		{UserID: "active-close", DistanceKm: 2, MaxDistanceKm: 50, LastActiveAt: now.Add(-time.Hour)},
	}
	ranked := r.Rank(now, candidates)
	want := []string{"active-close-shared", "active-close", "stale-far"}
	for i, id := range want {
		if ranked[i].UserID != id {
			t.Fatalf("position %d: want %s, got %s", i, id, ranked[i].UserID)
		}
	}
	// Input must not be mutated.
	if candidates[0].UserID != "stale-far" {
		t.Fatal("Rank must not reorder the input slice")
	}
}

func TestWeightedRankerIsDeterministicOnTies(t *testing.T) {
	now := time.Now()
	r := DefaultRanker()
	c := []Candidate{{UserID: "b", LastActiveAt: now}, {UserID: "a", LastActiveAt: now}}
	ranked := r.Rank(now, c)
	if ranked[0].UserID != "a" {
		t.Fatalf("ties must be broken by id, got %s first", ranked[0].UserID)
	}
}

func TestScoreBounds(t *testing.T) {
	now := time.Now()
	r := DefaultRanker()
	best := r.Score(now, Candidate{DistanceKm: 0, MaxDistanceKm: 10, LastActiveAt: now, SharedInterests: 10})
	if best < 0.99 || best > 1.01 {
		t.Fatalf("best score should be ~1, got %f", best)
	}
	worst := r.Score(now, Candidate{DistanceKm: 100, MaxDistanceKm: 10, LastActiveAt: now.Add(-1000 * time.Hour)})
	if worst < 0 || worst > 0.01 {
		t.Fatalf("worst score should be ~0, got %f", worst)
	}
}
