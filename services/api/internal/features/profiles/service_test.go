package profiles

import "testing"

func TestApproxDistanceBuckets(t *testing.T) {
	cases := []struct {
		km   float64
		want int
	}{{0, 1}, {0.4, 1}, {1, 1}, {1.2, 2}, {9.9, 10}, {10, 10}, {10.1, 15}, {47, 50}, {51, 55}}
	for _, tc := range cases {
		if got := ApproxDistance(tc.km); got != tc.want {
			t.Errorf("ApproxDistance(%v)=%d want %d", tc.km, got, tc.want)
		}
	}
}

func TestNormalizeLocationRoundsAndValidates(t *testing.T) {
	loc, err := normalizeLocation(&Location{Latitude: 45.76421, Longitude: 4.83569})
	if err != nil || loc.Latitude != 45.76 || loc.Longitude != 4.84 {
		t.Fatalf("expected 45.76/4.84, got %+v (%v)", loc, err)
	}
	if _, err := normalizeLocation(&Location{Latitude: 91, Longitude: 0}); err == nil {
		t.Fatal("latitude out of range must fail")
	}
	if got, err := normalizeLocation(nil); got != nil || err != nil {
		t.Fatal("nil location clears the stored location")
	}
}

func TestDedupeNormalizes(t *testing.T) {
	got := dedupe([]string{" Travel", "travel", "", "Music"})
	if len(got) != 2 || got[0] != "travel" || got[1] != "music" {
		t.Fatalf("unexpected %v", got)
	}
}
