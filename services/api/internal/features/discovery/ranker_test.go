package discovery

import (
	"math"
	"testing"
	"time"
)

func TestDefaultRankerOrdering(t *testing.T) {
	now := time.Now()
	items := []RankItem{
		{UserID: "far-stale", DistanceKm: 90, SharedInterests: 0, LastActiveAt: now.Add(-200 * time.Hour)},
		{UserID: "near-fresh", DistanceKm: 2, SharedInterests: 0, LastActiveAt: now},
		{UserID: "shared", DistanceKm: 30, SharedInterests: 3, LastActiveAt: now.Add(-24 * time.Hour)},
		{UserID: "unknown-dist", DistanceKm: math.NaN(), SharedInterests: 0, LastActiveAt: now.Add(-1 * time.Hour)},
	}
	got := DefaultRanker{}.Rank(now, items)
	if len(got) != len(items) {
		t.Fatalf("must return every item: %d", len(got))
	}
	if got[0].UserID != "shared" {
		t.Errorf("3 shared interests should win, got %s", got[0].UserID)
	}
	if got[len(got)-1].UserID != "far-stale" {
		t.Errorf("far and stale should be last, got %s", got[len(got)-1].UserID)
	}
	seen := map[string]bool{}
	for _, it := range got {
		if seen[it.UserID] {
			t.Errorf("duplicate %s", it.UserID)
		}
		seen[it.UserID] = true
	}
}

func TestDefaultRankerDoesNotMutateInputAndIsDeterministic(t *testing.T) {
	now := time.Now()
	in := []RankItem{
		{UserID: "a", DistanceKm: 10, LastActiveAt: now.Add(-time.Hour)},
		{UserID: "b", DistanceKm: 10, LastActiveAt: now.Add(-time.Hour)},
		{UserID: "c", DistanceKm: 1, LastActiveAt: now},
	}
	copyIn := append([]RankItem{}, in...)
	first := DefaultRanker{}.Rank(now, in)
	second := DefaultRanker{}.Rank(now, in)
	for i := range in {
		if in[i].UserID != copyIn[i].UserID {
			t.Fatal("input mutated")
		}
		if first[i].UserID != second[i].UserID {
			t.Fatal("not deterministic")
		}
	}
	if empty := (DefaultRanker{}).Rank(now, nil); empty == nil {
		t.Error("empty input returns an empty (non-nil) slice")
	}
}
