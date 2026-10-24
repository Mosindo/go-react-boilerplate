package geo

import (
	"math"
	"testing"
)

func TestHaversineParisLyon(t *testing.T) {
	d := Haversine(48.8566, 2.3522, 45.7640, 4.8357)
	if math.Abs(d-392) > 5 {
		t.Fatalf("expected ~392km, got %.1f", d)
	}
	if Haversine(10, 10, 10, 10) != 0 {
		t.Fatal("same point must be 0")
	}
}

func TestBucketKmNeverExposesExactDistance(t *testing.T) {
	cases := map[float64]int{0: 5, 0.2: 5, 5: 5, 5.1: 10, 12: 15, 392.4: 395}
	for in, want := range cases {
		if got := BucketKm(in); got != want {
			t.Errorf("BucketKm(%v)=%d want %d", in, got, want)
		}
	}
}

func TestCoarsenAndValidate(t *testing.T) {
	if got := Coarsen(48.856614); got != 48.86 {
		t.Fatalf("got %v", got)
	}
	if ValidCoordinates(91, 0) || ValidCoordinates(0, 181) || ValidCoordinates(math.NaN(), 0) {
		t.Fatal("expected invalid coordinates")
	}
	if !ValidCoordinates(-33.86, 151.2) {
		t.Fatal("expected valid")
	}
}
