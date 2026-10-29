package middleware

import (
	"strings"
	"testing"
)

func TestValidRequestID(t *testing.T) {
	for _, ok := range []string{"abc123", "req-1_2.3", strings.Repeat("a", 64)} {
		if !validRequestID(ok) {
			t.Errorf("%q should be accepted", ok)
		}
	}
	for _, bad := range []string{"", strings.Repeat("a", 65), "a b", "a\nb", `a"b`, "é"} {
		if validRequestID(bad) {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
