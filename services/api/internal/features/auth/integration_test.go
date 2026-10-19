package auth

import (
	"strings"
	"testing"
)

func TestCleanEmail(t *testing.T) {
	good := map[string]string{
		"  Alice@Example.COM ": "alice@example.com",
		"a.b+tag@sub.site.io":  "a.b+tag@sub.site.io",
	}
	for in, want := range good {
		if got, ok := cleanEmail(in); !ok || got != want {
			t.Errorf("cleanEmail(%q) = %q,%v want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "plain", "a@b", "Name <a@b.co>", "a@@b.co", "@b.co", "a b@c.co"} {
		if _, ok := cleanEmail(in); ok {
			t.Errorf("cleanEmail(%q) should be rejected", in)
		}
	}
}

func TestGenerateResetCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code, err := generateResetCode()
		if err != nil || len(code) != resetCodeLength {
			t.Fatalf("bad code %q: %v", code, err)
		}
		for _, r := range code {
			if !strings.ContainsRune(resetAlphabet, r) {
				t.Fatalf("code %q has char outside alphabet", code)
			}
		}
		seen[code] = true
	}
	if len(seen) < 49 {
		t.Fatal("codes must be random")
	}
}

func TestHashRefreshTokenIsStable(t *testing.T) {
	if hashRefreshToken("abc") != hashRefreshToken("abc") || hashRefreshToken("abc") == hashRefreshToken("abd") {
		t.Fatal("hash must be deterministic and collision-free for different inputs")
	}
	if hashRefreshToken("   ") != "" {
		t.Fatal("blank tokens must hash to empty so they never match a session")
	}
}
