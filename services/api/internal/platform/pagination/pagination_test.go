package pagination

import (
	"testing"
	"time"
)

func TestLimit(t *testing.T) {
	if n, err := Limit("", 10, 20); err != nil || n != 10 {
		t.Fatalf("default: %d %v", n, err)
	}
	if n, err := Limit("20", 10, 20); err != nil || n != 20 {
		t.Fatalf("max: %d %v", n, err)
	}
	for _, bad := range []string{"0", "-1", "21", "abc", "1.5"} {
		if _, err := Limit(bad, 10, 20); err == nil {
			t.Errorf("limit %q should fail", bad)
		}
	}
}

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	id := "11111111-1111-4111-8111-111111111111"
	got, err := Decode(Cursor{At: at, ID: id}.Encode())
	if err != nil || !got.At.Equal(at) || got.ID != id {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	if c, err := Decode(""); c != nil || err != nil {
		t.Fatal("empty cursor means first page")
	}
	for _, bad := range []string{"!!!", "bm90LWEtY3Vyc29y", Cursor{At: at, ID: "x'; DROP TABLE users;--"}.Encode()} {
		if _, err := Decode(bad); err == nil {
			t.Errorf("cursor %q must be rejected", bad)
		}
	}
}
