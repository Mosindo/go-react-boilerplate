package auth

import (
	"strings"
	"testing"
	"time"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func TestGenerateResetCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		c, err := GenerateResetCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != ResetCodeLength {
			t.Fatalf("length %d", len(c))
		}
		for _, r := range c {
			if !strings.ContainsRune(resetAlphabet, r) {
				t.Fatalf("code %q has a character outside the alphabet", c)
			}
		}
		seen[c] = true
	}
	if len(seen) < 495 {
		t.Errorf("codes are not random enough: %d unique of 500", len(seen))
	}
	if strings.ContainsAny(resetAlphabet, "01OI") {
		t.Error("alphabet must not contain look-alike characters")
	}
}

func TestResetCodeHashing(t *testing.T) {
	h := HashResetCode(testSecret, "user-1", "ABCD2345")
	if len(h) != 64 || strings.Contains(h, "ABCD") {
		t.Fatalf("hash %q", h)
	}
	if !ResetCodeMatches(testSecret, "user-1", "ABCD2345", h) {
		t.Error("correct code must match")
	}
	if !ResetCodeMatches(testSecret, "user-1", " abcd-2345 ", h) {
		t.Error("normalisation (case, dash, spaces) must apply")
	}
	if ResetCodeMatches(testSecret, "user-1", "ABCD2346", h) {
		t.Error("wrong code must not match")
	}
	if ResetCodeMatches(testSecret, "user-2", "ABCD2345", h) {
		t.Error("hash is bound to the user")
	}
	if ResetCodeMatches([]byte("another-secret-another-secret-12"), "user-1", "ABCD2345", h) {
		t.Error("hash is bound to the server secret")
	}
	if ResetCodeMatches(testSecret, "user-1", "ABCD2345", "") {
		t.Error("empty stored hash never matches")
	}
}

func TestResetUsable(t *testing.T) {
	now := time.Now()
	future, past := now.Add(time.Minute), now.Add(-time.Minute)
	cases := []struct {
		attempts int
		exp      time.Time
		used     bool
		want     bool
	}{
		{0, future, false, true},
		{4, future, false, true},
		{5, future, false, false},
		{0, past, false, false},
		{0, future, true, false},
	}
	for _, c := range cases {
		if got := ResetUsable(c.attempts, c.exp, c.used, now); got != c.want {
			t.Errorf("ResetUsable(%d, %v, %v) = %v", c.attempts, c.exp.Sub(now), c.used, got)
		}
	}
	if ResetCodeTTL != 30*time.Minute || ResetMaxAttempts != 5 {
		t.Error("policy constants changed")
	}
}

func TestPrehashHandlesLongPasswords(t *testing.T) {
	a := prehash(strings.Repeat("a", 100))
	b := prehash(strings.Repeat("a", 101))
	if len(a) > 72 || string(a) == string(b) {
		t.Error("prehash must fit bcrypt's limit and still distinguish long inputs")
	}
}
