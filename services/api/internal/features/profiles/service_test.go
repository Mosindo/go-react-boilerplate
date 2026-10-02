package profiles

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // turns 18 tomorrow
		{"2000-01-01", 26},
		{"2000-12-31", 25},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(b, now); got != c.want {
			t.Errorf("AgeOn(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestApproxDistance(t *testing.T) {
	cases := map[float64]int{0: 5, 0.2: 5, 5: 5, 5.1: 10, 12.3: 15, 99.9: 100}
	for in, want := range cases {
		if got := ApproxDistance(in); got != want {
			t.Errorf("ApproxDistance(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestRoundCoordIsCoarse(t *testing.T) {
	if got := roundCoord(48.856613); got != 48.86 {
		t.Fatalf("got %v", got)
	}
	if got := roundCoord(-2.3522219); got != -2.35 {
		t.Fatalf("got %v", got)
	}
}
