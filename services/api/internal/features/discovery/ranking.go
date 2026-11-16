package discovery

import (
	"hash/fnv"
	"math"
	"sort"
	"time"
)

// Ranker orders candidates. It is an interface so the recommendation strategy
// can evolve (collaborative filtering, ML scoring...) without touching the
// candidate generation, which enforces the hard rules.
type Ranker interface {
	Rank(viewerID string, candidates []Candidate, now time.Time) []Candidate
}

// HeuristicRanker scores candidates with transparent, explainable signals:
//   - reciprocity: people who already liked the viewer surface earlier, which
//     makes matches likelier without revealing who liked whom;
//   - shared interests;
//   - recent activity (people who are actually around);
//   - proximity;
//   - a small deterministic jitter per viewer and day so the deck does not
//     look identical every session while staying stable within a day.
type HeuristicRanker struct{}

func (HeuristicRanker) Rank(viewerID string, candidates []Candidate, now time.Time) []Candidate {
	type scored struct {
		c     Candidate
		score float64
	}
	day := now.UTC().Format("2006-01-02")
	items := make([]scored, len(candidates))
	for i, c := range candidates {
		items[i] = scored{c: c, score: Score(c, now) + jitter(viewerID, c.UserID, day)}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return items[i].c.UserID < items[j].c.UserID
		}
		return items[i].score > items[j].score
	})
	out := make([]Candidate, len(items))
	for i, it := range items {
		out[i] = it.c
	}
	return out
}

// Score returns the deterministic part of the ranking score.
func Score(c Candidate, now time.Time) float64 {
	score := 0.0
	if c.LikedViewer {
		score += 2.0
	}
	score += 1.5 * math.Min(float64(c.SharedInterests), 5) / 5

	hoursIdle := now.Sub(c.LastActiveAt).Hours()
	if hoursIdle < 0 {
		hoursIdle = 0
	}
	score += 1.5 * math.Exp(-hoursIdle/72)

	if c.DistanceKm != nil && c.MaxDistanceKm > 0 {
		closeness := 1 - math.Min(*c.DistanceKm/float64(c.MaxDistanceKm), 1)
		score += closeness
	}
	return score
}

func jitter(viewerID, candidateID, day string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(viewerID + "|" + candidateID + "|" + day))
	return float64(h.Sum32()%1000) / 1000 * 0.4
}
