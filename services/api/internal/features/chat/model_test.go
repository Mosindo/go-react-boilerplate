package chat

import (
	"strings"
	"testing"
)

func TestCleanBody(t *testing.T) {
	if b, err := CleanBody("  salut  "); err != nil || b != "salut" {
		t.Fatalf("got %q %v", b, err)
	}
	for _, bad := range []string{"", "   ", "a\x00b", strings.Repeat("é", MaxBodyRunes+1)} {
		if _, err := CleanBody(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
	if _, err := CleanBody(strings.Repeat("é", MaxBodyRunes)); err != nil {
		t.Fatalf("max length (in runes) must be accepted: %v", err)
	}
}
