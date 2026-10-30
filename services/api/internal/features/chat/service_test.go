package chat

import (
	"strings"
	"testing"
)

func TestNormalizeBody(t *testing.T) {
	if got, err := normalizeBody("  salut  "); err != nil || got != "salut" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if got, err := normalizeBody("ligne 1\nligne 2"); err != nil || got != "ligne 1\nligne 2" {
		t.Fatalf("newlines should be kept: %q err=%v", got, err)
	}
	for name, bad := range map[string]string{
		"empty":    "   ",
		"too long": strings.Repeat("a", MaxMessageRunes+1),
		"control":  "hi\x00there",
	} {
		if _, err := normalizeBody(bad); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
	if _, err := normalizeBody(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Fatalf("limit is in characters, not bytes: %v", err)
	}
}
