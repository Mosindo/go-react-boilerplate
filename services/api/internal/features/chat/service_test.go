package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeBody(t *testing.T) {
	got, err := NormalizeBody("  salut  ")
	if err != nil || got != "salut" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeBody("   \n "); !errors.Is(err, ErrEmptyMessage) {
		t.Fatal("blank accepted")
	}
	if _, err := NormalizeBody(strings.Repeat("é", 2001)); !errors.Is(err, ErrTooLong) {
		t.Fatal("too long accepted")
	}
	if _, err := NormalizeBody("a\x00b"); err == nil {
		t.Fatal("control char accepted")
	}
	if _, err := NormalizeBody(strings.Repeat("é", 2000)); err != nil {
		t.Fatal("2000 runes must be accepted")
	}
}
