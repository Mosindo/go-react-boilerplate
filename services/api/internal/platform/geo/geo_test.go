package geo

import "testing"

func TestRoundCoord(t *testing.T) {
	if got := RoundCoord(48.856614); got != 48.86 {
		t.Fatalf("got %v", got)
	}
	if got := RoundCoord(-2.35221); got != -2.35 {
		t.Fatalf("got %v", got)
	}
}

func TestBucketKm(t *testing.T) {
	cases := map[float64]int{0: 5, 0.2: 5, 5: 5, 5.1: 10, 12: 15, 47.9: 50, 50: 50}
	for in, want := range cases {
		if got := BucketKm(in); got != want {
			t.Errorf("BucketKm(%v)=%d want %d", in, got, want)
		}
	}
}

func TestValidLatLng(t *testing.T) {
	if !ValidLatLng(48, 2) || ValidLatLng(91, 0) || ValidLatLng(0, 181) {
		t.Fatal("unexpected validation result")
	}
}
