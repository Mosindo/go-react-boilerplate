package main

import (
	"net/http"
	"testing"
)

func TestHealth(t *testing.T) {
	env := newTestEnv(t)
	if res := env.do(http.MethodGet, "/health", "", nil, ""); res.Status != http.StatusOK {
		t.Fatalf("health: %d", res.Status)
	}
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	env := newTestEnv(t)
	for _, path := range []string{"/me", "/me/profile", "/discover", "/matches", "/conversations", "/notifications", "/blocks", "/likes/received", "/interests"} {
		if res := env.do(http.MethodGet, path, "", nil, ""); res.Status != http.StatusUnauthorized {
			t.Errorf("GET %s without token: got %d, want 401", path, res.Status)
		}
	}
	if res := env.do(http.MethodGet, "/me", "not-a-jwt", nil, ""); res.Status != http.StatusUnauthorized {
		t.Errorf("garbage token: got %d", res.Status)
	}
}
