package matching

import (
	"hash/fnv"
	"sort"
	"time"
)

// Ranker orders a candidate pool. The SQL layer enforces every hard rule
// (mutual preferences, blocks, already-swiped, distance); a Ranker only
// expresses soft preferences, so a smarter recommendation model can replace
// it without touching eligibility or safety logic.
type Ranker interface {
	Rank(viewerID string, candidates []Candidate, now time.Time) []Candidate
}

// ScoreRanker is a transparent weighted score:
//   - shared interests (strongest signal),
//   - people who already liked the viewer (raises match probability),
//   - recent activity (avoid showing dormant profiles),
//   - proximity,
//   - a small deterministic jitter (stable for one viewer during a day) so
//     the feed is not identical for everyone with equal scores.
type ScoreRanker struct{}

func (ScoreRanker) Score(viewerID string, c Candidate, now time.Time) float64 {
	score := 3.0 * float64(min(c.SharedInterests, 5))
	if c.LikedViewer {
		score += 2.5
	}
	switch idle := now.Sub(c.LastActiveAt); {
	case idle <= 24*time.Hour:
		score += 2
	case idle <= 7*24*time.Hour:
		score += 1
	case idle > 30*24*time.Hour:
		score -= 2
	}
	if c.DistanceKm != nil {
		switch d := *c.DistanceKm; {
		case d <= 5:
			score += 1.5
		case d <= 20:
			score += 1
		case d <= 50:
			score += 0.5
		}
	}
	return score + jitter(viewerID, c.UserID, now)
}

func (r ScoreRanker) Rank(viewerID string, candidates []Candidate, now time.Time) []Candidate {
	out := make([]Candidate, len(candidates))
	copy(out, candidates)
	scores := make(map[string]float64, len(out))
	for _, c := range out {
		scores[c.UserID] = r.Score(viewerID, c, now)
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

func jitter(viewerID, candidateID string, now time.Time) float64 {
	h := fnv.New32a()
	h.Write([]byte(viewerID + "|" + candidateID + "|" + now.UTC().Format("2006-01-02")))
	return float64(h.Sum32()%1000) / 1000.0
}
