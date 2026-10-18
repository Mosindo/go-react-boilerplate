package matching

import "sort"

// Ranker orders a pool of eligible candidates. The repository already filtered
// out everything that must never be shown (blocks, preferences, swipes), so a
// Ranker only decides *order* and can be replaced without touching SQL or the
// API: plug a smarter implementation into NewService to evolve recommendations.
type Ranker interface {
	Rank(pool []candidate) []candidate
}

// ProximityInterestRanker favours nearby people who share interests.
type ProximityInterestRanker struct{}

func (ProximityInterestRanker) Rank(pool []candidate) []candidate {
	out := append([]candidate(nil), pool...)
	score := func(c candidate) float64 {
		distancePenalty := c.DistanceKm / 20
		if distancePenalty > 10 {
			distancePenalty = 10
		}
		return float64(c.SharedInterests)*2 - distancePenalty
	}
	sort.SliceStable(out, func(i, j int) bool { return score(out[i]) > score(out[j]) })
	return out
}
