package discovery

import "testing"

func TestBucketDistance(t *testing.T) {
	cases := map[float64]int{0: 5, 0.2: 5, 5: 5, 5.01: 10, 12: 15, 49.9: 50, 50: 50, 51: 55}
	for km, want := range cases {
		if got := BucketDistance(km); got != want {
			t.Errorf("BucketDistance(%v)=%d want %d", km, got, want)
		}
	}
}
