package geo

import (
	"math"
	"testing"
)

func TestDistanceKm(t *testing.T) {
	// Paris -> Lyon is about 392 km as the crow flies.
	d := DistanceKm(48.8566, 2.3522, 45.7640, 4.8357)
	if math.Abs(d-392) > 5 {
		t.Fatalf("Paris-Lyon = %.1f km", d)
	}
	if DistanceKm(10, 10, 10, 10) != 0 {
		t.Fatal("same point must be 0")
	}
}

func TestDisplayNeverExact(t *testing.T) {
	cases := map[float64]int{0: 1, 0.2: 1, 1.2: 2, 9.01: 10, 49.5: 50, 52: 50, 53: 55, 392: 390}
	for in, want := range cases {
		if got := Display(in); got != want {
			t.Errorf("Display(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestRound2(t *testing.T) {
	if Round2(48.856614) != 48.86 || Round2(-2.3522) != -2.35 {
		t.Fatal("coordinates must be rounded to 2 decimals")
	}
}
