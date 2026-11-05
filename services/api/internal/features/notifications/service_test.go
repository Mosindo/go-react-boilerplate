package notifications

import (
	"strings"
	"testing"
)

func TestPreview(t *testing.T) {
	if got := Preview("  salut \n  toi  "); got != "salut toi" {
		t.Fatalf("got %q", got)
	}
	long := Preview(strings.Repeat("é", 200))
	if r := []rune(long); len(r) != 80 || r[79] != '…' {
		t.Fatalf("expected 80 runes with ellipsis, got %d", len(r))
	}
}
