package discovery

import (
	"math"
	"sort"
	"time"
)

// RankItem is the ranking-relevant projection of a candidate.
type RankItem struct {
	UserID          string
	DistanceKm      float64 // math.NaN() when unknown
	SharedInterests int
	LastActiveAt    time.Time
}

// Ranker orders a candidate pool. Implementations must be pure and fast: they
// run on at most a few hundred items per /discover request. Swap the default in
// main.go (e.g. for an ML or A/B ranker) without touching the rest of discovery.
type Ranker interface {
	// Rank returns the items best-first. It must return every input exactly once.
	Rank(now time.Time, items []RankItem) []RankItem
}

// DefaultRanker scores each candidate as
//
//	score = 10 * sharedInterests
//	      + 20 * exp(-hoursSinceLastActive / 48)   (recent activity, 0..20)
//	      - 0.3 * min(distanceKm, 100)             (closer is better)
//
// Higher is better; ties fall back to the most recently active. Unknown
// distances take no distance penalty. There is deliberately no paid boost.
type DefaultRanker struct{}

func (DefaultRanker) Rank(now time.Time, items []RankItem) []RankItem {
	type scored struct {
		item  RankItem
		score float64
	}
	list := make([]scored, len(items))
	for i, it := range items {
		hours := now.Sub(it.LastActiveAt).Hours()
		if hours < 0 {
			hours = 0
		}
		score := 10*float64(it.SharedInterests) + 20*math.Exp(-hours/48)
		if !math.IsNaN(it.DistanceKm) {
			score -= 0.3 * math.Min(it.DistanceKm, 100)
		}
		list[i] = scored{it, score}
	}
	sort.SliceStable(list, func(a, b int) bool {
		if list[a].score != list[b].score {
			return list[a].score > list[b].score
		}
		return list[a].item.LastActiveAt.After(list[b].item.LastActiveAt)
	})
	out := make([]RankItem, len(list))
	for i, s := range list {
		out[i] = s.item
	}
	return out
}
