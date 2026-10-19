package chat

import (
	"strings"
	"testing"
)

func TestNormalizeBody(t *testing.T) {
	if got, err := NormalizeBody("  salut\nça va ?\t"); err != nil || got != "salut\nça va ?" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeBody(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Fatalf("2000 runes allowed: %v", err)
	}
	for name, in := range map[string]string{
		"empty":    "",
		"blank":    " \n\t ",
		"too long": strings.Repeat("a", MaxMessageRunes+1),
		"nul":      "a\x00b",
		"escape":   "a\x1bb",
		"invalid":  "a\xffb",
	} {
		if _, err := NormalizeBody(in); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}
