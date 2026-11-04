package geo

import "testing"

func TestCoarsen(t *testing.T) {
	if got := Coarsen(48.856614); got != 48.86 {
		t.Fatalf("got %v", got)
	}
}

func TestDistanceParisLyon(t *testing.T) {
	d := DistanceKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 390 || d > 400 {
		t.Fatalf("unexpected distance %v", d)
	}
}

func TestApproxKm(t *testing.T) {
	cases := map[float64]int{0.2: 5, 4.9: 5, 7.4: 5, 7.6: 10, 23: 25, 396: 395}
	for in, want := range cases {
		if got := ApproxKm(in); got != want {
			t.Errorf("ApproxKm(%v)=%d want %d", in, got, want)
		}
	}
}

func TestValidLatLng(t *testing.T) {
	if ValidLatLng(91, 0) || ValidLatLng(0, 181) || !ValidLatLng(10, 10) {
		t.Fatal("validation wrong")
	}
}
