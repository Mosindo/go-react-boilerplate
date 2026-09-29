package discovery

import (
	"math"
	"sort"
)

// Ranker decides the order of discovery candidates. OrderBy returns a trusted, constant SQL
// ORDER BY body (columns: shared_count, distance_km, last_active_at, user_id) used to pick the
// overfetched page; Rank then orders the fetched candidates in Go and may use any signal.
type Ranker interface {
	OrderBy() string
	Rank(cands []Candidate)
}

// DefaultRanker: more shared interests first, then nearer, then most recently active,
// with the user id as deterministic tie-break.
type DefaultRanker struct{}

func (DefaultRanker) OrderBy() string {
	return "shared_count DESC, distance_km ASC, last_active_at DESC, user_id ASC"
}

func (DefaultRanker) Rank(c []Candidate) {
	sort.SliceStable(c, func(i, j int) bool {
		a, b := c[i], c[j]
		if a.SharedInterests != b.SharedInterests {
			return a.SharedInterests > b.SharedInterests
		}
		if a.DistanceKm != b.DistanceKm {
			return a.DistanceKm < b.DistanceKm
		}
		if !a.LastActiveAt.Equal(b.LastActiveAt) {
			return a.LastActiveAt.After(b.LastActiveAt)
		}
		return a.UserID < b.UserID
	})
}

// BucketDistance hides the exact distance: ceil to 1 km under 10 km, ceil to 5 km from 10 km
// upwards, never below 1.
func BucketDistance(km float64) int {
	if math.IsNaN(km) || km < 0 {
		km = 0
	}
	if km < 10 {
		b := int(math.Ceil(km))
		if b < 1 {
			b = 1
		}
		return b
	}
	return int(math.Ceil(km/5)) * 5
}
