package chat

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	c := Cursor{At: time.Date(2026, 3, 1, 12, 30, 0, 123456000, time.UTC), ID: "8f14e45f-ceea-467f-a8f4-6d1f2b0c9e11"}
	got, err := DecodeCursor(c.Encode())
	if err != nil || got == nil || !got.At.Equal(c.At) || got.ID != c.ID {
		t.Fatalf("round trip failed: %v %#v", err, got)
	}
	if got, err := DecodeCursor(""); err != nil || got != nil {
		t.Fatal("empty cursor should mean first page")
	}
	for _, bad := range []string{"%%%", "bm9waXBl", "MjAyNi0wMS0wMXxub3QtYS11dWlk"} {
		if _, err := DecodeCursor(bad); err == nil {
			t.Errorf("cursor %q should be rejected", bad)
		}
	}
}

func TestNormalizeBody(t *testing.T) {
	if got, err := normalizeBody("  salut\x00 toi \n "); err != nil || got != "salut toi" {
		t.Fatalf("unexpected %q %v", got, err)
	}
	if _, err := normalizeBody("   "); err != ErrEmptyMessage {
		t.Fatal("blank message must be rejected")
	}
	long := make([]rune, 2001)
	for i := range long {
		long[i] = 'é'
	}
	if _, err := normalizeBody(string(long)); err != ErrMessageTooLong {
		t.Fatal("message over 2000 characters must be rejected")
	}
}
