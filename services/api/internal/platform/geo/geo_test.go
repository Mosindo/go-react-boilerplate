package geo

import (
	"math"
	"testing"
)

func TestCoarsen(t *testing.T) {
	if got := Coarsen(48.856613); got != 48.86 {
		t.Fatalf("got %v", got)
	}
	if got := Coarsen(-2.354999); got != -2.35 {
		t.Fatalf("got %v", got)
	}
}

func TestDistanceKmParisLyon(t *testing.T) {
	d := DistanceKm(48.8566, 2.3522, 45.7640, 4.8357)
	if math.Abs(d-392) > 5 {
		t.Fatalf("expected ~392 km, got %v", d)
	}
}

func TestDisplayKmBuckets(t *testing.T) {
	cases := map[float64]int{0: 1, 0.4: 1, 1.6: 2, 7.4: 7, 19.4: 19, 22: 20, 23: 25, 391.7: 390}
	for in, want := range cases {
		if got := DisplayKm(in); got != want {
			t.Errorf("DisplayKm(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestBoundingBoxContainsRadius(t *testing.T) {
	minLat, maxLat, minLon, maxLon := BoundingBox(48.85, 2.35, 50)
	for _, p := range [][2]float64{{48.85 + 0.44, 2.35}, {48.85, 2.35 + 0.67}, {48.85 - 0.44, 2.35 - 0.67}} {
		if DistanceKm(48.85, 2.35, p[0], p[1]) <= 50 && (p[0] < minLat || p[0] > maxLat || p[1] < minLon || p[1] > maxLon) {
			t.Fatalf("point %v within radius is outside the box", p)
		}
	}
}
