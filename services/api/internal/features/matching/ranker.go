package matching

import (
	"math"
	"sort"
	"time"
)

// Ranker orders discovery candidates. Swap the implementation to experiment
// with other recommendation strategies without touching SQL or handlers.
type Ranker interface {
	Rank(candidates []Candidate, now time.Time) []Candidate
}

// DefaultRanker blends activity recency, shared interests, proximity and a
// small boost for people who already liked the viewer. Scores are additive
// and bounded so no single signal dominates.
type DefaultRanker struct{}

func (DefaultRanker) Score(c Candidate, now time.Time) float64 {
	days := now.Sub(c.LastActiveAt).Hours() / 24
	recency := math.Exp(-math.Max(0, days) / 7) // 1.0 now, ~0.37 after a week
	shared := math.Min(float64(c.SharedInterests), 5) / 5
	proximity := 0.5 // unknown distance is neutral
	if c.DistanceKm != nil {
		proximity = 1 / (1 + *c.DistanceKm/10)
	}
	liked := 0.0
	if c.LikedViewer {
		liked = 1
	}
	return 0.4*recency + 0.3*shared + 0.2*proximity + 0.1*liked
}

func (r DefaultRanker) Rank(candidates []Candidate, now time.Time) []Candidate {
	out := make([]Candidate, len(candidates))
	copy(out, candidates)
	scores := make(map[string]float64, len(out))
	for _, c := range out {
		scores[c.UserID] = r.Score(c, now)
	}
	sort.SliceStable(out, func(i, j int) bool { return scores[out[i].UserID] > scores[out[j].UserID] })
	return out
}
