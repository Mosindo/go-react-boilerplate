package messaging

import (
	"strings"
	"testing"
)

func TestCleanBody(t *testing.T) {
	ok := map[string]string{"  hello  ": "hello", "line1\nline2": "line1\nline2", "emoji 😀": "emoji 😀"}
	for in, want := range ok {
		got, err := CleanBody(in)
		if err != nil || got != want {
			t.Errorf("CleanBody(%q)=%q,%v want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "   \n ", "bell\x07", "nul\x00", strings.Repeat("x", 2001), "bad\xff\xfeutf8"} {
		if _, err := CleanBody(in); err == nil {
			t.Errorf("CleanBody(%q) should fail", in)
		}
	}
	if _, err := CleanBody(strings.Repeat("é", 2000)); err != nil {
		t.Errorf("2000 runes must be accepted: %v", err)
	}
}
