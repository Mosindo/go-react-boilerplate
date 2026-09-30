package chat

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 3, 1, 10, 0, 0, 123456000, time.UTC)
	id := "11111111-2222-3333-4444-555555555555"
	c, ok := decodeCursor(encodeCursor(at, id))
	if !ok || c == nil || !c.At.Equal(at) || c.ID != id {
		t.Fatalf("round trip failed: %+v %v", c, ok)
	}
	if c, ok := decodeCursor(""); !ok || c != nil {
		t.Fatal("empty cursor means first page")
	}
	for _, bad := range []string{"%%%", "bm90LWEtY3Vyc29y", encodeCursor(at, "not-a-uuid")} {
		if _, ok := decodeCursor(bad); ok {
			t.Errorf("cursor %q should be rejected", bad)
		}
	}
}
