package geo

import "testing"

func TestApproxKm(t *testing.T) {
	cases := map[float64]int{0: 1, 0.4: 1, 1.2: 2, 9.9: 10, 10.1: 15, 47: 50}
	for in, want := range cases {
		if got := ApproxKm(in); got != want {
			t.Errorf("ApproxKm(%v)=%d want %d", in, got, want)
		}
	}
}

func TestRoundCoordHidesPrecision(t *testing.T) {
	if got := RoundCoord(48.856613); got != 48.86 {
		t.Fatalf("got %v", got)
	}
}

func TestHaversineParisLyon(t *testing.T) {
	d := HaversineKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 390 || d > 400 {
		t.Fatalf("unexpected distance %v", d)
	}
}

func TestValidCoords(t *testing.T) {
	if ValidCoords(91, 0) || ValidCoords(0, 181) || !ValidCoords(10, 10) {
		t.Fatal("coordinate validation broken")
	}
}
