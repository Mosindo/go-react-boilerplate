package matching

import (
	"testing"

	"example.com/api/internal/features/profiles"
)

func c(id string, km float64, shared int) candidate {
	return candidate{Card: profiles.Card{UserID: id}, DistanceKm: km, SharedInterests: shared}
}

func TestRankerPrefersSharedInterestsThenProximity(t *testing.T) {
	pool := []candidate{c("far-many", 90, 4), c("near-none", 1, 0), c("near-some", 2, 2), c("far-none", 80, 0)}
	got := ProximityInterestRanker{}.Rank(pool)
	order := []string{got[0].Card.UserID, got[1].Card.UserID, got[2].Card.UserID, got[3].Card.UserID}
	want := []string{"near-some", "far-many", "near-none", "far-none"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
	if pool[0].Card.UserID != "far-many" {
		t.Fatal("Rank must not reorder the caller's slice")
	}
}

func TestCardOfOnlyExposesDistanceWhenAllowed(t *testing.T) {
	hidden := cardOf(candidate{Card: profiles.Card{UserID: "a"}, DistanceKm: 5, ShowDistance: false})
	if hidden.DistanceKm != nil {
		t.Fatal("distance must stay hidden when the target opted out")
	}
	shown := cardOf(candidate{Card: profiles.Card{UserID: "a"}, DistanceKm: 5, ShowDistance: true})
	if shown.DistanceKm == nil || *shown.DistanceKm != 5 {
		t.Fatal("distance expected")
	}
}
