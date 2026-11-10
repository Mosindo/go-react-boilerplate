package notifications

import "testing"

func TestTruncate(t *testing.T) {
	if got := truncate("  hello \n  world ", 80); got != "hello world" {
		t.Fatalf("got %q", got)
	}
	if got := truncate("ééééééééééé", 5); got != "éééé…" {
		t.Fatalf("got %q", got)
	}
}
