package matching

import (
	"math"
	"sort"
	"time"
)

// Ranker orders a discovery pool. It is the extension point for a smarter
// recommendation engine (collaborative filtering, ML scoring...): the
// repository only guarantees that every candidate is eligible.
type Ranker interface {
	Rank(now time.Time, maxDistanceKm int, pool []Candidate) []Candidate
}

// WeightedRanker blends proximity, recent activity, shared interests and
// reciprocal interest. Weights sum to 1 for readability.
type WeightedRanker struct {
	Proximity  float64
	Activity   float64
	Interests  float64
	Reciprocal float64
}

func DefaultRanker() WeightedRanker {
	return WeightedRanker{Proximity: 0.35, Activity: 0.3, Interests: 0.2, Reciprocal: 0.15}
}

func (w WeightedRanker) Score(now time.Time, maxDistanceKm int, c Candidate) float64 {
	proximity := 1.0
	if maxDistanceKm > 0 {
		proximity = 1 - math.Min(c.DistanceKm/float64(maxDistanceKm), 1)
	}
	// Activity decays with a 3-day half-life.
	days := math.Max(now.Sub(c.LastActiveAt).Hours()/24, 0)
	activity := math.Pow(0.5, days/3)
	interests := math.Min(float64(c.SharedInterests)/5, 1)
	reciprocal := 0.0
	if c.LikedViewer {
		reciprocal = 1
	}
	return w.Proximity*proximity + w.Activity*activity + w.Interests*interests + w.Reciprocal*reciprocal
}

func (w WeightedRanker) Rank(now time.Time, maxDistanceKm int, pool []Candidate) []Candidate {
	out := make([]Candidate, len(pool))
	copy(out, pool)
	scores := make(map[string]float64, len(out))
	for _, c := range out {
		scores[c.UserID] = w.Score(now, maxDistanceKm, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if scores[out[i].UserID] == scores[out[j].UserID] {
			return out[i].UserID < out[j].UserID
		}
		return scores[out[i].UserID] > scores[out[j].UserID]
	})
	return out
}
