package matching

import (
	"math/rand"
	"sort"
	"time"
)

// Ranker orders discovery candidates. The default implementation is a transparent,
// explainable score; it can be swapped for a learned model without touching the API.
type Ranker interface {
	Rank(candidates []Candidate, maxDistanceKm int, now time.Time) []Candidate
}

type ScoreRanker struct {
	// Jitter adds a little randomness so decks don't look identical between sessions.
	Jitter float64
	rng    *rand.Rand
}

func NewScoreRanker() *ScoreRanker {
	return &ScoreRanker{Jitter: 0.5, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// Score combines shared interests, recent activity, proximity and reciprocal interest.
func Score(c Candidate, maxDistanceKm int, now time.Time) float64 {
	score := 0.0
	shared := c.SharedInterests
	if shared > 5 {
		shared = 5
	}
	score += float64(shared) * 1.2

	inactive := now.Sub(c.LastActiveAt)
	switch {
	case inactive < 24*time.Hour:
		score += 3
	case inactive < 7*24*time.Hour:
		score += 1.5
	case inactive < 30*24*time.Hour:
		score += 0.5
	}

	if c.DistanceKm != nil && maxDistanceKm > 0 {
		closeness := 1 - *c.DistanceKm/float64(maxDistanceKm)
		if closeness > 0 {
			score += closeness * 2
		}
	}

	if c.LikedMe {
		score += 2.5
	}
	return score
}

func (r *ScoreRanker) Rank(candidates []Candidate, maxDistanceKm int, now time.Time) []Candidate {
	type scored struct {
		c Candidate
		s float64
	}
	items := make([]scored, len(candidates))
	for i, c := range candidates {
		s := Score(c, maxDistanceKm, now)
		if r.Jitter > 0 && r.rng != nil {
			s += r.rng.Float64() * r.Jitter
		}
		items[i] = scored{c: c, s: s}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].s > items[j].s })
	out := make([]Candidate, len(items))
	for i, it := range items {
		out[i] = it.c
	}
	return out
}
