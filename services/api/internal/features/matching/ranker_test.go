package matching

import (
	"testing"
	"time"
)

func TestScoreRankerPrefersAffinityActivityAndProximity(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	near, far := 2.0, 45.0
	viewer := ViewerContext{UserID: "viewer", MaxDistanceKm: 50, Now: now}

	strong := Candidate{UserID: "strong", DistanceKm: &near, SharedInterests: 4, LastActiveAt: now.Add(-time.Hour), PhotoCount: 4, HasBio: true}
	weak := Candidate{UserID: "weak", DistanceKm: &far, SharedInterests: 0, LastActiveAt: now.Add(-60 * 24 * time.Hour), PhotoCount: 1}
	likedMe := Candidate{UserID: "liked", DistanceKm: &far, SharedInterests: 0, LikedViewer: true, LastActiveAt: now.Add(-2 * time.Hour), PhotoCount: 1}

	ranked := ScoreRanker{}.Rank(viewer, []Candidate{weak, likedMe, strong})
	if ranked[0].UserID != "strong" || ranked[2].UserID != "weak" {
		t.Fatalf("unexpected order: %v, %v, %v", ranked[0].UserID, ranked[1].UserID, ranked[2].UserID)
	}
	if Score(viewer, likedMe) <= Score(viewer, weak) {
		t.Fatal("a profile that liked the viewer should rank above an equivalent one")
	}
}

func TestScoreIsDeterministicWithinADay(t *testing.T) {
	now := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	c := Candidate{UserID: "c", LastActiveAt: now}
	a := Score(ViewerContext{UserID: "v", Now: now}, c)
	b := Score(ViewerContext{UserID: "v", Now: now.Add(3 * time.Hour)}, c)
	if a != b {
		t.Fatal("jitter must be stable within a day")
	}
}

func TestOrderPair(t *testing.T) {
	a, b := orderPair("b", "a")
	if a != "a" || b != "b" {
		t.Fatal("pair must be ordered")
	}
}
