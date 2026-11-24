package matching

import (
	"testing"
	"time"
)

func TestOrderedPair(t *testing.T) {
	low, high := OrderedPair("b", "a")
	if low != "a" || high != "b" {
		t.Fatal("expected canonical order")
	}
	low2, high2 := OrderedPair("a", "b")
	if low != low2 || high != high2 {
		t.Fatal("pair must be symmetric")
	}
}

func TestWeightedRankerPrefersCloseActiveSharedAndReciprocal(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	far := Candidate{UserID: "far", DistanceKm: 45, SharedCount: 0, LastActiveAt: now.Add(-30 * 24 * time.Hour)}
	close := Candidate{UserID: "close", DistanceKm: 2, SharedCount: 3, LastActiveAt: now.Add(-time.Hour)}
	reciprocal := Candidate{UserID: "reciprocal", DistanceKm: 20, SharedCount: 1, LikedViewer: true, LastActiveAt: now.Add(-2 * time.Hour)}

	ranked := DefaultRanker().Rank([]Candidate{far, reciprocal, close}, 50, now)
	if ranked[len(ranked)-1].UserID != "far" {
		t.Fatalf("far inactive candidate should rank last, got %v", ranked)
	}
	if ranked[0].UserID != "close" && ranked[0].UserID != "reciprocal" {
		t.Fatalf("unexpected top candidate %v", ranked[0].UserID)
	}
}

func TestRankerIsDeterministicOnTies(t *testing.T) {
	now := time.Now()
	a := Candidate{UserID: "a", LastActiveAt: now}
	b := Candidate{UserID: "b", LastActiveAt: now}
	ranked := DefaultRanker().Rank([]Candidate{b, a}, 10, now)
	if ranked[0].UserID != "a" {
		t.Fatal("ties must be broken by user id")
	}
}
