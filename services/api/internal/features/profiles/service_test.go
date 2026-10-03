package profiles

import (
	"testing"
	"time"
)

func ptr(f float64) *float64 { return &f }

func TestApproxDistance(t *testing.T) {
	cases := []struct {
		in   *float64
		want *int
	}{
		{nil, nil},
		{ptr(0.2), intp(1)},
		{ptr(1), intp(1)},
		{ptr(3.2), intp(4)},
		{ptr(10), intp(10)},
		{ptr(12.4), intp(10)},
		{ptr(13), intp(15)},
		{ptr(387.6), intp(390)},
	}
	for _, c := range cases {
		got := ApproxDistance(c.in)
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Fatalf("ApproxDistance(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func intp(i int) *int { return &i }

func TestRoundCoordinateKeepsAbout1km(t *testing.T) {
	if got := RoundCoordinate(48.856614); got != 48.86 {
		t.Fatalf("got %v", got)
	}
	if got := RoundCoordinate(-2.349014); got != -2.35 {
		t.Fatalf("got %v", got)
	}
}

func TestAgeOn(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	for birth, want := range map[string]int{
		"2008-06-15": 18, // birthday today
		"2008-06-16": 17, // birthday tomorrow
		"1990-01-01": 36,
		"2000-12-31": 25,
	} {
		b, _ := time.Parse("2006-01-02", birth)
		if got := AgeOn(b, now); got != want {
			t.Fatalf("born %s: got %d want %d", birth, got, want)
		}
	}
}
