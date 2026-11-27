package chat

import (
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	got, err := NormalizeContent("  salut\x07 !\r\n ça va ?  ")
	if err != nil || got != "salut !\n ça va ?" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := NormalizeContent(" \n\t "); err == nil {
		t.Fatal("blank messages must be rejected")
	}
	if _, err := NormalizeContent(strings.Repeat("é", MaxMessageRunes+1)); err == nil {
		t.Fatal("too long messages must be rejected")
	}
	if _, err := NormalizeContent(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Fatalf("max length must be accepted: %v", err)
	}
}

func TestValidClientID(t *testing.T) {
	if !validClientID("") || !validClientID("abc-123_XYZ") {
		t.Fatal("expected valid ids")
	}
	if validClientID("a b") || validClientID(strings.Repeat("a", 65)) || validClientID("<x>") {
		t.Fatal("expected invalid ids")
	}
}
