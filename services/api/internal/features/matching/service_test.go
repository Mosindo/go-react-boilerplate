package matching

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	in := Cursor{CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC), ID: "0b8a6f8e-6a5e-4c52-9d0e-6d7a7a2a1c11"}
	out, err := decodeCursor(encodeCursor(in))
	if err != nil || !out.CreatedAt.Equal(in.CreatedAt) || out.ID != in.ID {
		t.Fatalf("round trip failed: %+v %v", out, err)
	}
	for _, bad := range []string{"", "!!!", encodeCursor(Cursor{CreatedAt: in.CreatedAt, ID: "x"})} {
		if _, err := decodeCursor(bad); err == nil {
			t.Errorf("cursor %q must be rejected", bad)
		}
	}
}

func TestApproxDistance(t *testing.T) {
	cases := map[float64]int{0.2: 5, 4.9: 5, 5.1: 10, 12: 15, 100: 100}
	for km, want := range cases {
		if got := *approxDistance(km); got != want {
			t.Errorf("approxDistance(%v) = %d, want %d", km, got, want)
		}
	}
}
