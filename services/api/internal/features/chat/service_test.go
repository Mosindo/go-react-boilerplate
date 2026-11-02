package chat

import (
	"strings"
	"testing"
)

func TestCleanBody(t *testing.T) {
	if got, err := CleanBody("  salut\nça va ?  "); err != nil || got != "salut\nça va ?" {
		t.Fatalf("%q %v", got, err)
	}
	for name, in := range map[string]string{
		"empty": "", "spaces": "  \n ", "too long": strings.Repeat("é", MaxMessageRunes+1), "nul": "a\x00b", "escape": "a\x1bb",
	} {
		if _, err := CleanBody(in); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := CleanBody(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Errorf("limit is in runes, not bytes: %v", err)
	}
}
