package discovery

import (
	"math"
	"sort"
	"time"
)

// Ranker orders candidates. The default implementation is a transparent,
// explainable weighted score; it can be swapped (A/B tests, ML model) without
// touching filtering or the API.
type Ranker interface {
	Rank(viewer Viewer, candidates []Candidate, now time.Time) []Candidate
}

type Weights struct {
	LikedViewer     float64
	SharedInterest  float64
	MaxShared       int
	Recency         float64
	RecencyHalfLife time.Duration
	Proximity       float64
	Completeness    float64
	Newcomer        float64
	NewcomerWindow  time.Duration
}

var DefaultWeights = Weights{
	LikedViewer:     40,
	SharedInterest:  6,
	MaxShared:       5,
	Recency:         25,
	RecencyHalfLife: 72 * time.Hour,
	Proximity:       15,
	Completeness:    8,
	Newcomer:        6,
	NewcomerWindow:  7 * 24 * time.Hour,
}

type ScoreRanker struct {
	Weights Weights
}

func NewScoreRanker() *ScoreRanker {
	return &ScoreRanker{Weights: DefaultWeights}
}

// Score computes a candidate's relevance for viewer.
func (r *ScoreRanker) Score(viewer Viewer, c Candidate, now time.Time) float64 {
	w := r.Weights
	score := 0.0
	if c.LikedViewer {
		score += w.LikedViewer
	}
	shared := c.SharedInterests
	if shared > w.MaxShared {
		shared = w.MaxShared
	}
	score += float64(shared) * w.SharedInterest

	inactive := now.Sub(c.LastActiveAt)
	if inactive < 0 {
		inactive = 0
	}
	score += w.Recency * math.Pow(0.5, inactive.Hours()/w.RecencyHalfLife.Hours())

	if c.DistanceKm != nil && viewer.MaxDistanceKm > 0 {
		ratio := *c.DistanceKm / float64(viewer.MaxDistanceKm)
		if ratio > 1 {
			ratio = 1
		}
		score += w.Proximity * (1 - ratio)
	}

	completeness := 0.0
	if c.HasBio {
		completeness += 0.5
	}
	photos := c.PhotoCount
	if photos > 3 {
		photos = 3
	}
	completeness += float64(photos) / 6
	score += w.Completeness * completeness

	if now.Sub(c.CreatedAt) < w.NewcomerWindow {
		score += w.Newcomer
	}
	return score
}

func (r *ScoreRanker) Rank(viewer Viewer, candidates []Candidate, now time.Time) []Candidate {
	type scored struct {
		c     Candidate
		score float64
	}
	items := make([]scored, len(candidates))
	for i, c := range candidates {
		items[i] = scored{c: c, score: r.Score(viewer, c, now)}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].c.UserID < items[j].c.UserID
	})
	out := make([]Candidate, len(items))
	for i, it := range items {
		out[i] = it.c
	}
	return out
}
