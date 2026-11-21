package geo

import (
	"math"
	"testing"
)

func TestCoarsenSnapsToGrid(t *testing.T) {
	lat, lng := Coarsen(48.856613, 2.352222)
	if math.Abs(lat-48.86) > 1e-9 || math.Abs(lng-2.35) > 1e-9 {
		t.Fatalf("unexpected coarse coordinates %v,%v", lat, lng)
	}
}

func TestDistanceKmParisLyon(t *testing.T) {
	d := DistanceKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("Paris-Lyon distance should be ~392km, got %v", d)
	}
}

func TestDisplayDistanceNeverZero(t *testing.T) {
	if DisplayDistanceKm(0.1) != 1 || DisplayDistanceKm(2.6) != 3 {
		t.Fatal("unexpected display distance")
	}
}

func TestValid(t *testing.T) {
	if !Valid(0, 0) || Valid(91, 0) || Valid(0, 181) || Valid(math.NaN(), 0) {
		t.Fatal("unexpected validity")
	}
}

func TestBoundingBoxContainsRadius(t *testing.T) {
	latDelta, lngDelta := BoundingBox(48.85, 50)
	if DistanceKm(48.85, 2.35, 48.85+latDelta, 2.35) < 49 {
		t.Fatal("lat delta too small")
	}
	if DistanceKm(48.85, 2.35, 48.85, 2.35+lngDelta) < 49 {
		t.Fatal("lng delta too small")
	}
}
