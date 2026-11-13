package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeMessage(t *testing.T) {
	if got, err := NormalizeMessage("  salut  "); err != nil || got != "salut" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeMessage(" \n\t "); !errors.Is(err, ErrMessageContentNeeded) {
		t.Fatal("blank message must be rejected")
	}
	if _, err := NormalizeMessage(strings.Repeat("é", 2000)); err != nil {
		t.Fatal("2000 runes is allowed")
	}
	if _, err := NormalizeMessage(strings.Repeat("é", 2001)); !errors.Is(err, ErrMessageTooLong) {
		t.Fatal("2001 runes must be rejected")
	}
}

func TestDirectConversationKeyIsSymmetric(t *testing.T) {
	a, b := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	if directConversationKey(a, b) != directConversationKey(b, a) {
		t.Fatal("conversation key must not depend on argument order")
	}
}
