package pagination

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	id := "0b9f2e1a-4c3d-4e5f-8a9b-0c1d2e3f4a5b"
	c, err := Decode(Encode(ts, id))
	if err != nil || c == nil || !c.Time.Equal(ts) || c.ID != id {
		t.Fatalf("round trip: %+v %v", c, err)
	}
	if c, err := Decode(""); c != nil || err != nil {
		t.Fatal("empty cursor must mean first page")
	}
	for _, bad := range []string{"!!!", "bm90LWEtY3Vyc29y", Encode(ts, "not-a-uuid")} {
		if _, err := Decode(bad); err == nil {
			t.Errorf("cursor %q accepted", bad)
		}
	}
}
