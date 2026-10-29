package chat

import (
	"strings"
	"testing"
)

func TestCleanBody(t *testing.T) {
	if got, err := cleanBody("  hello\r\nworld \n"); err != nil || got != "hello\nworld" {
		t.Fatalf("got %q, %v", got, err)
	}
	for name, in := range map[string]string{
		"empty":    "   \n\t",
		"too long": strings.Repeat("a", MaxMessageLength+1),
		"control":  "hi\x07there",
		"nul":      "a\x00b",
	} {
		if _, err := cleanBody(in); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if _, err := cleanBody(strings.Repeat("é", MaxMessageLength)); err != nil {
		t.Errorf("limit counts characters, not bytes: %v", err)
	}
}
