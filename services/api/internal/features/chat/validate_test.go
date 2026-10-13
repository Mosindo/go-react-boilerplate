package chat

import (
	"strings"
	"testing"
)

func TestNormalizeMessage(t *testing.T) {
	got, err := NormalizeMessage("  salut\u0000 toi ​!\r\nça va ? 👨‍👩‍👧 ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "salut toi !\nça va ? 👨‍👩‍👧" {
		t.Fatalf("unexpected normalization: %q", got)
	}
	for _, bad := range []string{"", "   \n\t ", "\u0000​", strings.Repeat("a", MaxMessageRunes+1)} {
		if _, err := NormalizeMessage(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
	if _, err := NormalizeMessage(strings.Repeat("é", MaxMessageRunes)); err != nil {
		t.Errorf("limit is in characters, not bytes: %v", err)
	}
}

func TestPreview(t *testing.T) {
	if got := Preview("a  b\nc", 10); got != "a b c" {
		t.Fatalf("got %q", got)
	}
	if got := Preview(strings.Repeat("x", 100), 10); len([]rune(got)) != 10 || !strings.HasSuffix(got, "…") {
		t.Fatalf("got %q", got)
	}
}
