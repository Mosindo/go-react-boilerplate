package chat

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCleanMessage(t *testing.T) {
	got, err := cleanMessage("  Salut\x00 !\x07\nça va ?  ")
	if err != nil || got != "Salut !\nça va ?" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "   ", "\x00\x01", strings.Repeat("é", MaxMessageRunes+1)} {
		if _, err := cleanMessage(bad); !errors.Is(err, ErrInvalidMessage) {
			t.Errorf("%q should be rejected, got %v", bad, err)
		}
	}
	if _, err := cleanMessage(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Fatalf("2000 characters are allowed: %v", err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	id := "11111111-2222-3333-4444-555555555555"
	cur, err := decodeCursor(encodeCursor(at, id))
	if err != nil || !cur.At.Equal(at) || cur.ID != id {
		t.Fatalf("round trip failed: %v %v", cur, err)
	}
	if cur, err := decodeCursor(""); cur != nil || err != nil {
		t.Fatal("empty cursor means first page")
	}
	for _, bad := range []string{"!!!", "bm90LWEtY3Vyc29y", encodeCursor(at, "short")} {
		if _, err := decodeCursor(bad); !errors.Is(err, ErrInvalidCursor) {
			t.Errorf("%q should be invalid, got %v", bad, err)
		}
	}
}
