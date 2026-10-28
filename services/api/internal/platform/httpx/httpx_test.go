package httpx

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	id := "11111111-2222-3333-4444-555555555555"
	got, err := DecodeCursor(EncodeCursor(at, id))
	if err != nil || got == nil || !got.At.Equal(at) || got.ID != id {
		t.Fatalf("round trip failed: %+v %v", got, err)
	}
	if c, err := DecodeCursor(""); c != nil || err != nil {
		t.Fatal("empty cursor means first page")
	}
	for _, bad := range []string{"%%%", "bm9waXBl", EncodeCursor(at, "not-a-uuid")} {
		if _, err := DecodeCursor(bad); err == nil {
			t.Errorf("cursor %q should be rejected", bad)
		}
	}
}
