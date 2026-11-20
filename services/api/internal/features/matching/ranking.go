package matching

import (
	"math"
	"sort"
)

// Ranker orders discovery candidates. It is an interface so the heuristic
// can later be replaced (collaborative filtering, learned model...) without
// touching the repository or the HTTP layer.
type Ranker interface {
	Rank(candidates []Candidate, viewer ViewerContext) []Candidate
}

// HeuristicRanker blends activity, proximity, shared interests and
// reciprocity. Weights are deliberately simple and documented.
type HeuristicRanker struct{}

const (
	weightRecency    = 0.35
	weightProximity  = 0.25
	weightInterests  = 0.25
	weightReciprocal = 0.15
)

func (HeuristicRanker) Score(c Candidate, viewer ViewerContext) float64 {
	recency := 1 / (1 + math.Max(0, c.IdleHours)/24)

	proximity := 0.5
	if c.DistanceKm != nil && viewer.MaxDistanceKm > 0 {
		proximity = math.Max(0, 1-*c.DistanceKm/float64(viewer.MaxDistanceKm))
	}

	interests := math.Min(float64(c.SharedInterests), 5) / 5

	reciprocal := 0.0
	if c.LikedViewer {
		reciprocal = 1
	}

	return weightRecency*recency + weightProximity*proximity + weightInterests*interests + weightReciprocal*reciprocal
}

func (r HeuristicRanker) Rank(candidates []Candidate, viewer ViewerContext) []Candidate {
	ranked := append([]Candidate(nil), candidates...)
	scores := make(map[string]float64, len(ranked))
	for _, c := range ranked {
		scores[c.UserID] = r.Score(c, viewer)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		si, sj := scores[ranked[i].UserID], scores[ranked[j].UserID]
		if si != sj {
			return si > sj
		}
		return ranked[i].UserID < ranked[j].UserID
	})
	return ranked
}
