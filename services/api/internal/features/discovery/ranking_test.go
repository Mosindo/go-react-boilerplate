package discovery

import "testing"

func TestApproximateDistance(t *testing.T) {
	cases := []struct {
		km   float64
		want int
	}{
		{0.0, 1}, {0.4, 1}, {1.0, 1}, {1.2, 2}, {9.4, 10}, {9.99, 10},
		{10.0, 10}, {12.4, 10}, {12.6, 15}, {47.0, 45}, {48.0, 50}, {203.0, 205},
	}
	for _, tc := range cases {
		if got := approximateDistance(tc.km); got != tc.want {
			t.Errorf("approximateDistance(%v) = %d, want %d", tc.km, got, tc.want)
		}
	}
}
