package chat

import (
	"strings"
	"testing"
)

func TestCleanBody(t *testing.T) {
	if v, err := cleanBody("  hello\nworld  "); err != nil || v != "hello\nworld" {
		t.Fatalf("unexpected: %q %v", v, err)
	}
	for name, in := range map[string]string{"empty": "  ", "control": "a\x07b", "too long": strings.Repeat("é", MaxBodyLength+1)} {
		if _, err := cleanBody(in); err == nil {
			t.Errorf("%s must be refused", name)
		}
	}
	if _, err := cleanBody(strings.Repeat("é", MaxBodyLength)); err != nil {
		t.Fatalf("exactly the max length must pass: %v", err)
	}
}
