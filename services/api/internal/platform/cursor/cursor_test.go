package cursor

import (
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 123456000, time.UTC)
	id := "8f14e45f-ceea-467f-a0f6-4d3a2c7b9e10"
	pos, err := Decode(Encode(now, id))
	if err != nil || pos.ID != id || !pos.Time.Equal(now) {
		t.Fatalf("round trip failed: %+v %v", pos, err)
	}
	if pos, err := Decode(""); pos != nil || err != nil {
		t.Fatal("empty cursor must mean first page")
	}
	for _, bad := range []string{"!!", "bm9waXBl", Encode(now, "short")} {
		if _, err := Decode(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}
