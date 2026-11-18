package people

import (
	"testing"
	"time"
)

func TestAge(t *testing.T) {
	birth := time.Date(2000, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := map[string]int{
		"2018-06-14": 17,
		"2018-06-15": 18,
		"2025-01-01": 24,
	}
	for day, want := range cases {
		now, _ := time.Parse("2006-01-02", day)
		if got := Age(birth, now); got != want {
			t.Fatalf("Age at %s = %d, want %d", day, got, want)
		}
	}
}

func TestRoundCoordinate(t *testing.T) {
	if got := RoundCoordinate(48.856613); got != 48.86 {
		t.Fatalf("unexpected rounding: %v", got)
	}
	if got := RoundCoordinate(-2.354999); got != -2.35 {
		t.Fatalf("unexpected rounding: %v", got)
	}
}

func TestDisplayDistanceKm(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{{0.2, 1}, {3.4, 3}, {9.6, 10}, {12.4, 10}, {13, 15}, {101, 100}}
	for _, tc := range cases {
		if got := DisplayDistanceKm(tc.in); got != tc.want {
			t.Fatalf("DisplayDistanceKm(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestHaversineKm(t *testing.T) {
	// Paris -> Lyon is roughly 392 km.
	d := HaversineKm(48.8566, 2.3522, 45.7640, 4.8357)
	if d < 385 || d > 400 {
		t.Fatalf("unexpected distance %v", d)
	}
}
