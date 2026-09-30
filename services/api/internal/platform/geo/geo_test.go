package geo

import (
	"math"
	"testing"
)

func TestCoarsenCoordinateRoundsToTwoDecimals(t *testing.T) {
	if got := CoarsenCoordinate(48.856613); got != 48.86 {
		t.Fatalf("expected 48.86, got %v", got)
	}
	if got := CoarsenCoordinate(-2.294481); got != -2.29 {
		t.Fatalf("expected -2.29, got %v", got)
	}
}

func TestHaversineParisLyon(t *testing.T) {
	d := HaversineKm(48.8566, 2.3522, 45.7640, 4.8357)
	if math.Abs(d-392) > 5 {
		t.Fatalf("expected ~392 km, got %.1f", d)
	}
}

func TestApproximateDistanceNeverRevealsExactValue(t *testing.T) {
	cases := map[float64]int{0: 2, 0.4: 2, 1.9: 2, 3.4: 3, 9.6: 10, 12: 10, 13: 15, 47: 45, 48: 50}
	for in, want := range cases {
		if got := ApproximateDistanceKm(in); got != want {
			t.Errorf("ApproximateDistanceKm(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestValidCoordinates(t *testing.T) {
	if ValidLatitude(91) || ValidLongitude(-181) || ValidLatitude(math.NaN()) {
		t.Fatal("expected invalid coordinates to be rejected")
	}
	if !ValidLatitude(-90) || !ValidLongitude(180) {
		t.Fatal("expected boundary coordinates to be accepted")
	}
}
