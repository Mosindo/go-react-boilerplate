package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewRateLimiter(60, time.Minute, 3)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d should pass within burst", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 {
		t.Fatal("4th request should be limited with a retry delay")
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be independent")
	}

	now = now.Add(time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("one token should be refilled after a second")
	}
}

func TestParseAccessTokenRejectsGarbage(t *testing.T) {
	if _, err := ParseAccessToken([]byte("secret"), "not-a-jwt"); err == nil {
		t.Fatal("expected error")
	}
	// alg=none token
	none := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOiJ4Iiwib2lkIjoieSIsInNpZCI6InoiLCJleHAiOjQxMDI0NDQ4MDB9."
	if _, err := ParseAccessToken([]byte("secret"), none); err == nil {
		t.Fatal("alg=none must be rejected")
	}
}
