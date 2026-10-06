package matching

import (
	"math"
	"time"
)

// Ranker orders discovery candidates. Swap the implementation to experiment
// with recommendation algorithms without touching filtering, SQL or HTTP code.
type Ranker interface {
	Score(v Viewer, c Candidate, sharedInterests int, now time.Time) float64
}

// HeuristicRanker is the default, explainable scoring:
//   - people who already liked the viewer come first (a like back is a match),
//   - shared interests,
//   - proximity relative to the viewer's maximum distance,
//   - recent activity (decays over two weeks).
type HeuristicRanker struct{}

func (HeuristicRanker) Score(v Viewer, c Candidate, shared int, now time.Time) float64 {
	score := 0.0
	if c.LikedMe {
		score += 3
	}
	score += math.Min(float64(shared), 5) * 0.6

	if c.DistanceKm != nil && v.Prefs.MaxDistanceKm > 0 {
		score += 1.5 * math.Max(0, 1-*c.DistanceKm/float64(v.Prefs.MaxDistanceKm))
	}

	idle := now.Sub(c.LastActiveAt).Hours() / 24
	score += 1.0 * math.Max(0, 1-idle/14)
	return score
}
