package chat

import (
	"strings"
	"testing"
)

func TestCleanBody(t *testing.T) {
	got, err := CleanBody("  salut\r\n\x00toi  ")
	if err != nil || got != "salut\ntoi" {
		t.Fatalf("got %q err %v", got, err)
	}
	for _, bad := range []string{"", "   \n ", strings.Repeat("a", MaxMessageLength+1)} {
		if _, err := CleanBody(bad); err == nil {
			t.Errorf("%q should be rejected", bad[:min(len(bad), 10)])
		}
	}
	if _, err := CleanBody(strings.Repeat("é", MaxMessageLength)); err != nil {
		t.Error("2000 runes must be accepted")
	}
}
