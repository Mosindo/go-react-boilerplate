package discovery

import (
	"math"
	"testing"
)

func TestDistanceBucket(t *testing.T) {
	cases := map[float64]int{0: 5, 0.2: 5, 5: 5, 5.01: 10, 9.9: 10, 10: 10, 10.1: 15, 49.9: 50, 123: 125}
	for in, want := range cases {
		if got := DistanceBucket(in); got != want {
			t.Errorf("DistanceBucket(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestHaversine(t *testing.T) {
	// Paris -> London is about 344 km.
	d := haversine(48.8566, 2.3522, 51.5074, -0.1278)
	if math.Abs(d-344) > 3 {
		t.Errorf("Paris-London = %.1f km", d)
	}
	if haversine(10, 10, 10, 10) != 0 {
		t.Error("same point must be 0")
	}
}
