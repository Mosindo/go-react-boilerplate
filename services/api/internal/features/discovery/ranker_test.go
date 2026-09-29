package discovery

import (
	"testing"
	"time"
)

func TestBucketDistance(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{
		{0, 1}, {0.2, 1}, {1, 1}, {1.01, 2}, {9.9, 10}, {10, 10}, {10.1, 15}, {14.9, 15},
		{15, 15}, {47, 50}, {500, 500}, {-3, 1},
	}
	for _, c := range cases {
		if got := BucketDistance(c.in); got != c.want {
			t.Errorf("BucketDistance(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestDefaultRankerOrdering(t *testing.T) {
	now := time.Now()
	c := []Candidate{
		{UserID: "e", SharedInterests: 1, DistanceKm: 1, LastActiveAt: now},
		{UserID: "d", SharedInterests: 2, DistanceKm: 9, LastActiveAt: now},
		{UserID: "b", SharedInterests: 2, DistanceKm: 3, LastActiveAt: now.Add(-time.Hour)},
		{UserID: "a", SharedInterests: 2, DistanceKm: 3, LastActiveAt: now},
		{UserID: "c", SharedInterests: 2, DistanceKm: 3, LastActiveAt: now},
	}
	DefaultRanker{}.Rank(c)
	got := ""
	for _, x := range c {
		got += x.UserID
	}
	if got != "acbde" {
		t.Fatalf("order = %s, want acbde", got)
	}
}

func TestBoundingBox(t *testing.T) {
	minLat, maxLat, minLon, maxLon, ok := boundingBox(48.85, 2.35, 50)
	if !ok || minLat >= 48.85 || maxLat <= 48.85 || minLon >= 2.35 || maxLon <= 2.35 {
		t.Fatalf("unexpected box %v %v %v %v %v", minLat, maxLat, minLon, maxLon, ok)
	}
	if _, _, _, _, ok := boundingBox(0, 179.9, 100); ok {
		t.Fatal("antimeridian must disable the longitude filter")
	}
	if _, _, _, _, ok := boundingBox(89.9, 0, 50); ok {
		t.Fatal("poles must disable the longitude filter")
	}
}

func TestParseLimit(t *testing.T) {
	for raw, want := range map[string]int{"": 10, "1": 1, "20": 20} {
		if got, err := parseLimit(raw); err != nil || got != want {
			t.Errorf("parseLimit(%q) = %d, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"0", "21", "-1", "abc", "1.5"} {
		if _, err := parseLimit(raw); err == nil {
			t.Errorf("parseLimit(%q) should fail", raw)
		}
	}
}
