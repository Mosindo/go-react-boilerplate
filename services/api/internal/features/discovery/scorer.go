package discovery

import (
	"math"
	"sort"
	"time"
)

// Scorer ranks candidates. It is an interface so the recommendation logic can
// evolve (A/B tests, learned models) without touching the retrieval query.
type Scorer interface {
	Score(c Candidate, now time.Time) float64
}

// DefaultScorer blends a few transparent signals:
//   - reciprocity: people who already liked the viewer come first, which
//     turns likes into matches faster (and is free for everyone);
//   - shared interests;
//   - proximity;
//   - recent activity, so dormant accounts sink;
//   - a small boost for new members so they get initial visibility.
type DefaultScorer struct{}

func (DefaultScorer) Score(c Candidate, now time.Time) float64 {
	score := 0.0
	if c.LikedViewer {
		score += 4
	}
	score += math.Min(float64(c.SharedInterests), 5) * 0.6
	if c.DistanceKm != nil {
		score += 2 / (1 + *c.DistanceKm/10)
	}
	inactiveHours := now.Sub(c.LastActiveAt).Hours()
	score += 2 / (1 + math.Max(inactiveHours, 0)/24)
	if now.Sub(c.CreatedAt) < 7*24*time.Hour {
		score += 0.5
	}
	return score
}

// Rank orders candidates by descending score, deterministic on ties.
func Rank(candidates []Candidate, scorer Scorer, now time.Time) []Candidate {
	type scored struct {
		c     Candidate
		score float64
	}
	items := make([]scored, len(candidates))
	for i, c := range candidates {
		items[i] = scored{c: c, score: scorer.Score(c, now)}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].c.UserID < items[j].c.UserID
	})
	out := make([]Candidate, len(items))
	for i, it := range items {
		out[i] = it.c
	}
	return out
}
