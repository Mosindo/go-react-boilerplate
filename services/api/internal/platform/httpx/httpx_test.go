package httpx

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	id := "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	c, err := DecodeCursor(EncodeCursor(at, id))
	if err != nil || !c.At.Equal(at) || c.ID != id {
		t.Fatalf("round trip failed: %+v %v", c, err)
	}
	if c, err := DecodeCursor(""); c != nil || err != nil {
		t.Fatal("empty cursor must be nil")
	}
	for _, bad := range []string{"!!!", "Zm9v", EncodeCursor(at, "not-a-uuid")} {
		if _, err := DecodeCursor(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("3f2504e0-4f89-11d3-9a0c-0305e82c3301") || IsUUID("3f2504e0-4f89-11d3-9a0c-0305e82c330") || IsUUID("../../etc/passwd/xxxxxxxxxxxxxxxxxxxx") {
		t.Fatal("IsUUID is wrong")
	}
}
