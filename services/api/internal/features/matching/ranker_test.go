package matching

import (
	"testing"
	"time"

	"example.com/api/internal/features/profiles"
)

func f(v float64) *float64 { return &v }

func TestHeuristicRankerPrefersLikesInterestsProximityAndActivity(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	viewer := Viewer{Prefs: profiles.Preferences{MaxDistanceKm: 50}}
	r := HeuristicRanker{}

	base := Candidate{DistanceKm: f(25), LastActiveAt: now.Add(-7 * 24 * time.Hour)}
	baseScore := r.Score(viewer, base, 0, now)

	liked := base
	liked.LikedMe = true
	if r.Score(viewer, liked, 0, now) <= baseScore {
		t.Error("a candidate who liked the viewer must rank higher")
	}
	if r.Score(viewer, base, 3, now) <= baseScore {
		t.Error("shared interests must raise the score")
	}
	near := base
	near.DistanceKm = f(2)
	if r.Score(viewer, near, 0, now) <= baseScore {
		t.Error("closer candidates must rank higher")
	}
	fresh := base
	fresh.LastActiveAt = now
	if r.Score(viewer, fresh, 0, now) <= baseScore {
		t.Error("recently active candidates must rank higher")
	}
	stale := base
	stale.LastActiveAt = now.Add(-90 * 24 * time.Hour)
	if r.Score(viewer, stale, 0, now) >= baseScore {
		t.Error("long-inactive candidates must rank lower")
	}
}
