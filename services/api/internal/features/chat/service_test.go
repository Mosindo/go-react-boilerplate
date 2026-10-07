package chat

import (
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	got, err := normalizeContent("  hello \n")
	if err != nil || got != "hello" {
		t.Fatalf("got %q, %v", got, err)
	}
	for name, in := range map[string]string{
		"empty":    "",
		"blank":    " \n\t ",
		"too long": strings.Repeat("é", maxMessageRunes+1),
		"nul":      "hi\x00there",
		"escape":   "hi\x1bthere",
	} {
		if _, err := normalizeContent(in); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, err := normalizeContent(strings.Repeat("é", maxMessageRunes)); err != nil {
		t.Errorf("exact limit should pass: %v", err)
	}
}
