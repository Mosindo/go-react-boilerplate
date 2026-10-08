package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/api/internal/server"
	"example.com/api/internal/testutil"
)

// Boots the full application graph against a migrated database and checks /health.
func TestApplicationBootsAndReportsHealthy(t *testing.T) {
	pool := testutil.NewPool(t)
	app, err := server.New(server.Options{DB: pool, JWTSecret: []byte("0123456789abcdef0123456789abcdef"), UploadDir: t.TempDir()})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	rec := httptest.NewRecorder()
	app.Engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
