package pagination

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	id := "27254b54-52e9-47c3-992f-eefd66ebab78"
	gotAt, gotID, err := Decode(Encode(at, id))
	if err != nil || !gotAt.Equal(at) || gotID != id {
		t.Fatalf("round trip failed: %v %v %v", gotAt, gotID, err)
	}
	for _, bad := range []string{"", "%%%", "bm90LWEtY3Vyc29y", Encode(at, "not-a-uuid")} {
		if _, _, err := Decode(bad); err == nil {
			t.Errorf("cursor %q should be rejected", bad)
		}
	}
}

func TestLimit(t *testing.T) {
	if Limit(0, 10, 50) != 10 || Limit(-3, 10, 50) != 10 || Limit(500, 10, 50) != 50 || Limit(7, 10, 50) != 7 {
		t.Fatal("limit clamping broken")
	}
}
