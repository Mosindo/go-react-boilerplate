package httpx

import (
	"testing"
	"time"
)

func TestTimeCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 678901234, time.UTC)
	c, ok := DecodeTimeCursor(EncodeTimeCursor(at, "abc"))
	if !ok || c == nil || !c.At.Equal(at) || c.ID != "abc" {
		t.Fatalf("round trip failed: %+v %v", c, ok)
	}
	if c, ok := DecodeTimeCursor(""); !ok || c != nil {
		t.Fatal("empty cursor means first page")
	}
	for _, bad := range []string{"!!!", EncodeCursor("no-separator"), EncodeCursor("not-a-time|id")} {
		if _, ok := DecodeTimeCursor(bad); ok {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
