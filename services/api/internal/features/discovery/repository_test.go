package discovery

import "testing"

func TestApproxKm(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	cases := []struct {
		in   *float64
		want int
	}{
		{f(0.2), 1}, {f(1.01), 2}, {f(9.4), 10}, {f(12), 10}, {f(13), 15}, {f(387.6), 390},
	}
	for _, c := range cases {
		got := ApproxKm(c.in)
		if got == nil || *got != c.want {
			t.Errorf("ApproxKm(%v) = %v, want %d", *c.in, got, c.want)
		}
	}
	if ApproxKm(nil) != nil {
		t.Error("nil distance must stay nil")
	}
}
