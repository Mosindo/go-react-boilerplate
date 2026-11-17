package discovery

import (
	"hash/fnv"
	"math"
	"sort"
	"time"
)

// Ranker orders eligible candidates. Swap the implementation to evolve the
// recommendation algorithm (collaborative filtering, ML scores...) without touching
// eligibility rules or the API.
type Ranker interface {
	Rank(viewerID string, candidates []Candidate, maxDistanceKm int, now time.Time) []Candidate
}

// WeightedRanker is a transparent heuristic: shared interests, proximity, recent
// activity, a boost for people who already liked the viewer, and a small daily
// deterministic jitter so the feed does not feel static.
type WeightedRanker struct {
	InterestWeight  float64
	ProximityWeight float64
	ActivityWeight  float64
	LikedYouBoost   float64
	JitterWeight    float64
}

func DefaultRanker() WeightedRanker {
	return WeightedRanker{InterestWeight: 1.0, ProximityWeight: 2.0, ActivityWeight: 2.0, LikedYouBoost: 1.5, JitterWeight: 0.75}
}

func (r WeightedRanker) Score(viewerID string, c Candidate, maxDistanceKm int, now time.Time) float64 {
	interests := math.Min(float64(c.SharedInterests), 5) * r.InterestWeight

	proximity := 0.0
	if maxDistanceKm > 0 {
		proximity = (1 - math.Min(c.DistanceKm/float64(maxDistanceKm), 1)) * r.ProximityWeight
	}

	daysInactive := now.Sub(c.LastActiveAt).Hours() / 24
	if daysInactive < 0 {
		daysInactive = 0
	}
	activity := math.Exp(-daysInactive/7) * r.ActivityWeight

	liked := 0.0
	if c.LikedViewer {
		liked = r.LikedYouBoost
	}

	return interests + proximity + activity + liked + jitter(viewerID, c.UserID, now)*r.JitterWeight
}

func (r WeightedRanker) Rank(viewerID string, candidates []Candidate, maxDistanceKm int, now time.Time) []Candidate {
	ranked := append([]Candidate(nil), candidates...)
	scores := make(map[string]float64, len(ranked))
	for _, c := range ranked {
		scores[c.UserID] = r.Score(viewerID, c, maxDistanceKm, now)
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

// jitter returns a stable pseudo-random value in [0,1) per viewer, candidate and day.
func jitter(viewerID, candidateID string, now time.Time) float64 {
	h := fnv.New64a()
	h.Write([]byte(viewerID))
	h.Write([]byte{0})
	h.Write([]byte(candidateID))
	h.Write([]byte(now.UTC().Format("2006-01-02")))
	return float64(h.Sum64()%10_000) / 10_000
}

// BoundingBoxAround computes a coarse lat/lng box used to prune candidates with an index
// before the exact great-circle distance is evaluated.
func BoundingBoxAround(lat, lng float64, radiusKm int) BoundingBox {
	latDelta := float64(radiusKm) / 111.0
	box := BoundingBox{MinLat: lat - latDelta, MaxLat: lat + latDelta}
	cosLat := math.Cos(lat * math.Pi / 180)
	if cosLat < 0.01 {
		return box // near the poles: latitude pruning only
	}
	lngDelta := float64(radiusKm) / (111.0 * cosLat)
	if lng-lngDelta < -180 || lng+lngDelta > 180 {
		return box // box crosses the antimeridian: skip longitude pruning
	}
	box.MinLng, box.MaxLng, box.CheckLongitude = lng-lngDelta, lng+lngDelta, true
	return box
}
