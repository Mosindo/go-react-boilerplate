package geo

import "testing"

func TestRoundCoordinate(t *testing.T) {
	if got := RoundCoordinate(48.85661); got != 48.86 {
		t.Fatalf("got %v", got)
	}
	if got := RoundCoordinate(-2.34999); got != -2.35 {
		t.Fatalf("got %v", got)
	}
}

func TestDistanceKm(t *testing.T) {
	// Paris - Lyon is about 392 km.
	d := DistanceKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("unexpected distance %v", d)
	}
	if DistanceKm(10, 10, 10, 10) != 0 {
		t.Fatal("same point must be 0")
	}
}

func TestDisplayDistance(t *testing.T) {
	cases := map[float64]int{0: 5, 0.4: 5, 5: 5, 5.1: 10, 12: 15, 392: 395}
	for km, want := range cases {
		if got := DisplayDistance(km); got != want {
			t.Fatalf("DisplayDistance(%v)=%d want %d", km, got, want)
		}
	}
}
