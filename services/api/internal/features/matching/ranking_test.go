package matching

import (
	"testing"
	"time"
)

func ptr(v float64) *float64 { return &v }

func TestHeuristicRankerPrefersRelevantProfiles(t *testing.T) {
	viewer := ViewerContext{MaxDistanceKm: 50}
	candidates := []Candidate{
		{UserID: "far-idle", DistanceKm: ptr(48), IdleHours: 24 * 30},
		{UserID: "close-active-shared", DistanceKm: ptr(2), IdleHours: 1, SharedInterests: 4},
		{UserID: "mid", DistanceKm: ptr(20), IdleHours: 12, SharedInterests: 1},
	}
	ranked := HeuristicRanker{}.Rank(candidates, viewer)
	if ranked[0].UserID != "close-active-shared" || ranked[2].UserID != "far-idle" {
		t.Fatalf("unexpected order: %v, %v, %v", ranked[0].UserID, ranked[1].UserID, ranked[2].UserID)
	}
}

func TestHeuristicRankerReciprocityBoost(t *testing.T) {
	viewer := ViewerContext{MaxDistanceKm: 50}
	a := Candidate{UserID: "a", DistanceKm: ptr(10), IdleHours: 2}
	b := a
	b.UserID = "b"
	b.LikedViewer = true
	ranked := HeuristicRanker{}.Rank([]Candidate{a, b}, viewer)
	if ranked[0].UserID != "b" {
		t.Fatal("a candidate who already liked the viewer should rank first at equal relevance")
	}
}

func TestHeuristicRankerIsDeterministic(t *testing.T) {
	viewer := ViewerContext{MaxDistanceKm: 50}
	same := []Candidate{{UserID: "z"}, {UserID: "a"}, {UserID: "m"}}
	ranked := HeuristicRanker{}.Rank(same, viewer)
	if ranked[0].UserID != "a" || ranked[1].UserID != "m" || ranked[2].UserID != "z" {
		t.Fatal("ties must be broken by user id")
	}
}

func TestAgeBounds(t *testing.T) {
	min, max := birthdateBounds(refDay(), 25, 30)
	// Someone born exactly 25 years ago is 25 (included); 31 years ago is too old.
	if !min.Before(max) {
		t.Fatal("lower bound must be before upper bound")
	}
	if max.Format("2006-01-02") != "2001-06-15" || min.Format("2006-01-02") != "1995-06-15" {
		t.Fatalf("unexpected bounds %v %v", min, max)
	}
}

func refDay() time.Time { return time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC) }
