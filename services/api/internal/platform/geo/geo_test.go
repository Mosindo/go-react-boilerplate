package geo

import (
	"math"
	"testing"
)

func TestRound(t *testing.T) {
	if got := Round(48.856613); math.Abs(got-48.86) > 1e-9 {
		t.Fatalf("Round = %v", got)
	}
	if got := Round(2.352222); math.Abs(got-2.35) > 1e-9 {
		t.Fatalf("Round = %v", got)
	}
}

func TestDistanceKmParisLyon(t *testing.T) {
	d := DistanceKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("unexpected Paris-Lyon distance %v", d)
	}
}

func TestApproximateKm(t *testing.T) {
	cases := map[float64]int{0.2: 1, 1.9: 1, 4.4: 4, 12.4: 10, 13: 15, 47: 45, 123: 120}
	for in, want := range cases {
		if got := ApproximateKm(in); got != want {
			t.Fatalf("ApproximateKm(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestValidCoordinates(t *testing.T) {
	if !ValidCoordinates(0, 0) || ValidCoordinates(91, 0) || ValidCoordinates(0, -181) || ValidCoordinates(math.NaN(), 0) {
		t.Fatal("ValidCoordinates mismatch")
	}
}

func TestBoundingBoxContainsRadius(t *testing.T) {
	minLat, maxLat, minLng, maxLng := BoundingBox(48.8566, 2.3522, 50)
	if DistanceKm(48.8566, 2.3522, minLat, 2.3522) < 49 || DistanceKm(48.8566, 2.3522, 48.8566, maxLng) < 49 {
		t.Fatal("bounding box smaller than radius")
	}
	if maxLat <= minLat || maxLng <= minLng {
		t.Fatal("invalid box")
	}
}
