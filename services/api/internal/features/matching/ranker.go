package matching

import (
	"math"
	"sort"
	"time"
)

// Ranker orders discovery candidates. The default implementation is a simple
// weighted score; a learned or collaborative model can replace it without
// touching SQL or handlers.
type Ranker interface {
	Rank(now time.Time, candidates []Candidate) []Candidate
}

type WeightedRanker struct {
	RecencyWeight   float64
	ProximityWeight float64
	InterestsWeight float64
	// RecencyHalfLife controls how fast inactive profiles sink.
	RecencyHalfLife time.Duration
}

func DefaultRanker() WeightedRanker {
	return WeightedRanker{
		RecencyWeight:   0.45,
		ProximityWeight: 0.35,
		InterestsWeight: 0.20,
		RecencyHalfLife: 72 * time.Hour,
	}
}

func (r WeightedRanker) Score(now time.Time, c Candidate) float64 {
	hours := now.Sub(c.LastActiveAt).Hours()
	if hours < 0 {
		hours = 0
	}
	recency := math.Pow(0.5, hours/r.RecencyHalfLife.Hours())

	proximity := 1.0
	if c.MaxDistanceKm > 0 {
		proximity = 1 - math.Min(c.DistanceKm/c.MaxDistanceKm, 1)
	}

	interests := math.Min(float64(c.SharedInterests), 5) / 5

	return r.RecencyWeight*recency + r.ProximityWeight*proximity + r.InterestsWeight*interests
}

func (r WeightedRanker) Rank(now time.Time, candidates []Candidate) []Candidate {
	out := make([]Candidate, len(candidates))
	copy(out, candidates)
	scores := make(map[string]float64, len(out))
	for _, c := range out {
		scores[c.UserID] = r.Score(now, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := scores[out[i].UserID], scores[out[j].UserID]
		if si != sj {
			return si > sj
		}
		return out[i].UserID < out[j].UserID
	})
	return out
}
