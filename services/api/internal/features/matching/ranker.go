package matching

import (
	"hash/fnv"
	"math"
	"sort"
)

// Ranker orders eligible candidates. The default is a transparent weighted
// score; a learned model or collaborative filtering can replace it without
// touching SQL or handlers.
type Ranker interface {
	Rank(viewer ViewerContext, candidates []Candidate) []Candidate
}

type ScoreRanker struct{}

func (ScoreRanker) Rank(viewer ViewerContext, candidates []Candidate) []Candidate {
	ranked := make([]Candidate, len(candidates))
	copy(ranked, candidates)
	scores := make(map[string]float64, len(ranked))
	for _, c := range ranked {
		scores[c.UserID] = Score(viewer, c)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return scores[ranked[i].UserID] > scores[ranked[j].UserID]
	})
	return ranked
}

// Score combines activity, affinity, proximity and profile quality. A small
// deterministic jitter (stable for a viewer during a day) avoids showing the
// exact same order to everyone.
func Score(viewer ViewerContext, c Candidate) float64 {
	score := 0.0

	inactive := viewer.Now.Sub(c.LastActiveAt).Hours()
	switch {
	case inactive <= 24:
		score += 3
	case inactive <= 24*7:
		score += 2
	case inactive <= 24*30:
		score += 1
	}

	score += 1.5 * math.Min(float64(c.SharedInterests), 5)

	if c.LikedViewer {
		score += 4
	}

	if c.DistanceKm != nil && viewer.MaxDistanceKm > 0 {
		ratio := math.Min(*c.DistanceKm/float64(viewer.MaxDistanceKm), 1)
		score += 2 * (1 - ratio)
	}

	if c.PhotoCount >= 3 {
		score += 1
	}
	if c.HasBio {
		score += 0.5
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(viewer.UserID + c.UserID + viewer.Now.Format("2006-01-02")))
	score += float64(h.Sum32()%1000) / 1000

	return score
}
