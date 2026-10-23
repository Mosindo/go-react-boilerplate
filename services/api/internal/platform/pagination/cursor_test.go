package pagination

import (
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	got, id, ok := Decode(Encode(now, "abc"))
	if !ok || !got.Equal(now) || id != "abc" {
		t.Fatalf("round trip failed: %v %q %v", got, id, ok)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"", "!!!", "bm9wZQ", "MjAyNnx4"} {
		if _, _, ok := Decode(bad); ok {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
