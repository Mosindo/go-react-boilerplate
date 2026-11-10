package discovery

import (
	"math"
	"sort"
	"time"
)

// Ranking weights. The recommendation strategy is isolated here so it can
// evolve (learning-to-rank, collaborative filtering...) without touching the
// SQL eligibility rules, which are the hard safety/preference constraints.
const (
	weightLikedViewer = 2.5  // reciprocity: surfacing people who already like you creates matches
	weightProximity   = 1.0  // closer is better, relative to the viewer's radius
	weightActivity    = 1.0  // recently active people are more likely to answer
	weightPerInterest = 0.3  // shared interests, capped
	maxInterestBonus  = 1.2  //
	activityHalfLife  = 72.0 // hours
	maxJitter         = 0.25 // keeps decks from being strictly identical between sessions
)

// Score computes the ranking score of one candidate. jitter must be in [0,1).
func Score(c Candidate, maxDistanceKm int, now time.Time, jitter float64) float64 {
	score := 0.0
	if c.LikedViewer {
		score += weightLikedViewer
	}
	if maxDistanceKm > 0 {
		score += weightProximity * math.Max(0, 1-c.DistanceKm/float64(maxDistanceKm))
	}
	hours := math.Max(0, now.Sub(c.LastActiveAt).Hours())
	score += weightActivity * math.Exp2(-hours/activityHalfLife)
	score += math.Min(maxInterestBonus, weightPerInterest*float64(c.SharedInterests))
	score += maxJitter * jitter
	return score
}

// Rank keeps candidates within the viewer radius, scores and orders them.
func Rank(cands []Candidate, maxDistanceKm int, now time.Time, jitter func() float64, limit int) []Candidate {
	kept := cands[:0]
	for _, c := range cands {
		if c.DistanceKm > float64(maxDistanceKm) {
			continue
		}
		c.Score = Score(c, maxDistanceKm, now, jitter())
		kept = append(kept, c)
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Score > kept[j].Score })
	if len(kept) > limit {
		kept = kept[:limit]
	}
	return kept
}
