package profiles

import (
	"testing"
	"time"
)

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow: still 17
		{"2000-01-01", 26},
		{"1996-12-31", 29},
		{"2004-02-29", 22}, // leap-day birthday
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeAt(b, now); got != c.want {
			t.Errorf("AgeAt(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestApproxDistanceNeverUnderstatesAndIsCoarse(t *testing.T) {
	cases := []struct {
		km   float64
		want int
	}{{0, 1}, {0.4, 1}, {1, 1}, {1.2, 2}, {9.9, 10}, {10, 10}, {10.1, 15}, {12, 15}, {15, 15}, {16, 20}, {101, 105}}
	for _, c := range cases {
		if got := ApproxDistance(c.km); got != c.want {
			t.Errorf("ApproxDistance(%v) = %d, want %d", c.km, got, c.want)
		}
	}
}

func TestRoundCoordKeepsTwoDecimals(t *testing.T) {
	cases := map[float64]float64{48.8566: 48.86, 2.3522: 2.35, -73.9857: -73.99, 0.004: 0, -0.006: -0.01}
	for in, want := range cases {
		if got := roundCoord(in); got != want {
			t.Errorf("roundCoord(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("3f2b8c1e-9d4a-4e6b-8a1f-0c2d3e4f5a6b") {
		t.Error("valid uuid rejected")
	}
	for _, bad := range []string{"", "abc", "3f2b8c1e-9d4a-4e6b-8a1f-0c2d3e4f5a6", "3f2b8c1e-9d4a-4e6b-8a1f-0c2d3e4f5a6b'; DROP TABLE users;--"} {
		if IsUUID(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}
