package chat

import (
	"strings"
	"testing"
)

func TestCleanBody(t *testing.T) {
	if got, err := cleanBody("  salut\nça va ? "); err != nil || got != "salut\nça va ?" {
		t.Errorf("got %q %v", got, err)
	}
	for name, in := range map[string]string{"empty": "  \n ", "long": strings.Repeat("é", 2001), "bell": "a\x07b", "invalid utf8": "\xff"} {
		if _, err := cleanBody(in); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
	if _, err := cleanBody(strings.Repeat("é", 2000)); err != nil {
		t.Error("2000 runes is the limit and must pass")
	}
}
