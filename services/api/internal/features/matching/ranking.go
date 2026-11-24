package matching

import (
	"math"
	"sort"
	"time"
)

// Ranker orders a candidate pool. The default implementation is a transparent
// weighted score; it can be swapped for a learned model without touching the
// repository or the handlers.
type Ranker interface {
	Rank(candidates []Candidate, maxDistanceKm int, now time.Time) []Candidate
}

type WeightedRanker struct {
	SharedInterestWeight float64
	ProximityWeight      float64
	ActivityWeight       float64
	ReciprocityWeight    float64
}

// DefaultRanker favours people who share interests, live close, were active
// recently, and gives a mild boost to people who already liked the viewer so
// that mutual interest surfaces sooner.
func DefaultRanker() WeightedRanker {
	return WeightedRanker{SharedInterestWeight: 1.0, ProximityWeight: 2.0, ActivityWeight: 1.5, ReciprocityWeight: 1.5}
}

func (r WeightedRanker) Score(c Candidate, maxDistanceKm int, now time.Time) float64 {
	shared := math.Min(float64(c.SharedCount), 5) / 5
	proximity := 1 - math.Min(c.DistanceKm/math.Max(float64(maxDistanceKm), 1), 1)
	hoursIdle := now.Sub(c.LastActiveAt).Hours()
	activity := math.Exp(-math.Max(hoursIdle, 0) / (24 * 7))
	score := r.SharedInterestWeight*shared + r.ProximityWeight*proximity + r.ActivityWeight*activity
	if c.LikedViewer {
		score += r.ReciprocityWeight
	}
	return score
}

func (r WeightedRanker) Rank(candidates []Candidate, maxDistanceKm int, now time.Time) []Candidate {
	ranked := append([]Candidate(nil), candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		si, sj := r.Score(ranked[i], maxDistanceKm, now), r.Score(ranked[j], maxDistanceKm, now)
		if si != sj {
			return si > sj
		}
		return ranked[i].UserID < ranked[j].UserID
	})
	return ranked
}

// OrderedPair returns the canonical (low, high) ordering stored in matches.
func OrderedPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
