package discovery

import (
	"testing"
	"time"
)

func TestWeightedRankerPrefersRelevantCandidates(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	r := DefaultRanker()
	r.JitterWeight = 0 // deterministic ordering for the assertion

	candidates := []Candidate{
		{UserID: "far-inactive", DistanceKm: 45, LastActiveAt: now.Add(-30 * 24 * time.Hour)},
		{UserID: "close-active-shared", DistanceKm: 2, LastActiveAt: now.Add(-time.Hour), SharedInterests: 3},
		{UserID: "liked-me", DistanceKm: 20, LastActiveAt: now.Add(-24 * time.Hour), LikedViewer: true},
	}
	ranked := r.Rank("viewer", candidates, 50, now)
	if ranked[0].UserID != "close-active-shared" || ranked[2].UserID != "far-inactive" {
		t.Fatalf("unexpected ranking: %+v", ranked)
	}
	if len(candidates) != 3 || candidates[0].UserID != "far-inactive" {
		t.Fatal("Rank must not mutate its input")
	}
}

func TestJitterIsStablePerDay(t *testing.T) {
	now := time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC)
	a := jitter("v", "c", now)
	if a != jitter("v", "c", now.Add(3*time.Hour)) {
		t.Fatal("jitter must be stable within a day")
	}
	if a < 0 || a >= 1 {
		t.Fatalf("jitter out of range: %v", a)
	}
}

func TestBoundingBoxAround(t *testing.T) {
	box := BoundingBoxAround(48.85, 2.35, 50)
	if !box.CheckLongitude || box.MinLat >= 48.85 || box.MaxLng <= 2.35 {
		t.Fatalf("unexpected box %+v", box)
	}
	if BoundingBoxAround(10, 179.9, 50).CheckLongitude {
		t.Fatal("antimeridian boxes must skip longitude pruning")
	}
	if BoundingBoxAround(89.99, 0, 50).CheckLongitude {
		t.Fatal("polar boxes must skip longitude pruning")
	}
}
