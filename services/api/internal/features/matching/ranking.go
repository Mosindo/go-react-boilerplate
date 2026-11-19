package matching

import (
	"math"
	"sort"
	"time"
)

// Candidate is a discovery candidate with the signals used for ranking.
type Candidate struct {
	UserID          string
	DistanceKm      *float64
	LastActiveAt    time.Time
	SharedInterests int
	LikedViewer     bool
}

// Ranker orders candidates. It is an interface so the recommendation strategy can
// evolve (collaborative filtering, ML scoring...) without touching the rest of the flow.
type Ranker interface {
	Rank(viewer Viewer, candidates []Candidate, now time.Time) []Candidate
}

// DefaultRanker blends a few transparent signals:
//   - someone who already liked you is surfaced sooner (helps reciprocity, free for everyone);
//   - shared interests;
//   - recent activity (inactive profiles sink);
//   - proximity within the chosen radius.
type DefaultRanker struct{}

func (DefaultRanker) Rank(viewer Viewer, candidates []Candidate, now time.Time) []Candidate {
	scored := make([]struct {
		c     Candidate
		score float64
	}, len(candidates))
	for i, c := range candidates {
		scored[i].c = c
		scored[i].score = Score(viewer, c, now)
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].c.UserID < scored[j].c.UserID
		}
		return scored[i].score > scored[j].score
	})
	out := make([]Candidate, len(scored))
	for i := range scored {
		out[i] = scored[i].c
	}
	return out
}

// Score is exported for unit tests and future tuning.
func Score(viewer Viewer, c Candidate, now time.Time) float64 {
	score := 0.0
	if c.LikedViewer {
		score += 2.0
	}
	score += 0.25 * math.Min(float64(c.SharedInterests), 4)

	inactiveDays := now.Sub(c.LastActiveAt).Hours() / 24
	if inactiveDays < 0 {
		inactiveDays = 0
	}
	score += 1.0 / (1.0 + inactiveDays/3.0)

	if c.DistanceKm != nil && viewer.MaxDistanceKm > 0 {
		score -= 0.5 * math.Min(*c.DistanceKm/float64(viewer.MaxDistanceKm), 1)
	}
	return score
}
