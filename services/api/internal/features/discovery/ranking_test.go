package discovery

import (
	"testing"
	"time"
)

func TestScorePrefersReciprocityInterestsAndActivity(t *testing.T) {
	now := time.Now()
	base := Candidate{UserID: "a", LastActiveAt: now.Add(-24 * time.Hour)}

	liked := base
	liked.LikedViewer = true
	if Score(liked, now) <= Score(base, now) {
		t.Fatal("a candidate who liked the viewer should score higher")
	}

	shared := base
	shared.SharedInterests = 3
	if Score(shared, now) <= Score(base, now) {
		t.Fatal("shared interests should increase the score")
	}

	stale := base
	stale.LastActiveAt = now.Add(-30 * 24 * time.Hour)
	if Score(stale, now) >= Score(base, now) {
		t.Fatal("inactive candidates should score lower")
	}

	near, far := 2.0, 45.0
	a, b := base, base
	a.DistanceKm, a.MaxDistanceKm = &near, 50
	b.DistanceKm, b.MaxDistanceKm = &far, 50
	if Score(a, now) <= Score(b, now) {
		t.Fatal("closer candidates should score higher")
	}
}

func TestRankIsDeterministicAndComplete(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cands := []Candidate{
		{UserID: "1", LastActiveAt: now},
		{UserID: "2", LastActiveAt: now, LikedViewer: true},
		{UserID: "3", LastActiveAt: now.Add(-90 * 24 * time.Hour)},
	}
	r := HeuristicRanker{}
	first := r.Rank("viewer", cands, now)
	second := r.Rank("viewer", cands, now)
	if len(first) != 3 {
		t.Fatal("ranking must not drop candidates")
	}
	for i := range first {
		if first[i].UserID != second[i].UserID {
			t.Fatal("ranking must be stable for a given viewer and day")
		}
	}
	if first[0].UserID != "2" {
		t.Fatalf("expected reciprocal like first, got %s", first[0].UserID)
	}
	if first[2].UserID != "3" {
		t.Fatalf("expected inactive profile last, got %s", first[2].UserID)
	}
}
