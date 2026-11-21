package matching

import (
	"errors"
	"testing"
)

func TestNormalizeAction(t *testing.T) {
	for raw, want := range map[string]string{"like": "like", " PASS ": "pass"} {
		got, err := NormalizeAction(raw)
		if err != nil || got != want {
			t.Fatalf("NormalizeAction(%q) = %q, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"", "superlike", "boost"} {
		if _, err := NormalizeAction(raw); !errors.Is(err, ErrInvalidAction) {
			t.Fatalf("expected invalid action for %q", raw)
		}
	}
}

func TestOrderedPair(t *testing.T) {
	a, b := orderedPair("b", "a")
	if a != "a" || b != "b" {
		t.Fatal("pair must be ordered")
	}
}
