package notifications

import "testing"

func TestTruncate(t *testing.T) {
	if Truncate("court", 10) != "court" {
		t.Fatal("short text must be unchanged")
	}
	if got := Truncate("ééééééééééé", 5); got != "éééé…" {
		t.Fatalf("unexpected %q", got)
	}
}
