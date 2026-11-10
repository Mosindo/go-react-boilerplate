package discovery

import (
	"testing"
	"time"
)

func noJitter() float64 { return 0 }

func TestRankFiltersByDistanceAndOrders(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	cands := []Candidate{
		{UserID: "far", DistanceKm: 80, LastActiveAt: now},
		{UserID: "near-inactive", DistanceKm: 1, LastActiveAt: now.Add(-30 * 24 * time.Hour)},
		{UserID: "near-active", DistanceKm: 2, LastActiveAt: now.Add(-time.Hour)},
		{UserID: "liked-me", DistanceKm: 40, LastActiveAt: now.Add(-48 * time.Hour), LikedViewer: true},
		{UserID: "shared", DistanceKm: 2, LastActiveAt: now.Add(-time.Hour), SharedInterests: 3},
	}
	got := Rank(cands, 50, now, noJitter, 10)
	if len(got) != 4 {
		t.Fatalf("expected far candidate to be filtered out, got %d", len(got))
	}
	order := []string{"liked-me", "shared", "near-active", "near-inactive"}
	for i, id := range order {
		if got[i].UserID != id {
			t.Fatalf("position %d: expected %s, got %s", i, id, got[i].UserID)
		}
	}
	if limited := Rank(cands[:3], 50, now, noJitter, 1); len(limited) != 1 {
		t.Fatal("limit must be applied")
	}
}

func TestScoreInterestBonusIsCapped(t *testing.T) {
	now := time.Now()
	a := Score(Candidate{SharedInterests: 4, LastActiveAt: now}, 50, now, 0)
	b := Score(Candidate{SharedInterests: 40, LastActiveAt: now}, 50, now, 0)
	if a != b {
		t.Fatalf("interest bonus must be capped: %v vs %v", a, b)
	}
}

func TestParseExclude(t *testing.T) {
	ids, err := ParseExclude(" 8F14E45F-CEEA-467F-A0F6-4D3A2C7B9E10 ,8f14e45f-ceea-467f-a0f6-4d3a2c7b9e11")
	if err != nil || len(ids) != 2 || ids[0] != "8f14e45f-ceea-467f-a0f6-4d3a2c7b9e10" {
		t.Fatalf("unexpected %v %v", ids, err)
	}
	if _, err := ParseExclude("1;DROP TABLE users"); err == nil {
		t.Fatal("expected invalid ids to be rejected")
	}
	if ids, err := ParseExclude(""); ids != nil || err != nil {
		t.Fatal("empty exclude must be nil")
	}
}
