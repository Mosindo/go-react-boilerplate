package chat

import (
	"strings"
	"testing"
)

func TestNormalizeBody(t *testing.T) {
	if _, err := NormalizeBody("  \n "); err == nil {
		t.Fatal("blank must fail")
	}
	if got, err := NormalizeBody("  salut\nça va ?  "); err != nil || got != "salut\nça va ?" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeBody(strings.Repeat("a", MaxMessageRunes+1)); err == nil {
		t.Fatal("too long must fail")
	}
	if _, err := NormalizeBody("a\x07b"); err == nil {
		t.Fatal("control characters must fail")
	}
}
