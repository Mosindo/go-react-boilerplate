package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func corsRouter(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders(), CORS(origins))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })
	r.OPTIONS("/x", func(c *gin.Context) { c.String(200, "ok") })
	return r
}

func do(r http.Handler, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/x", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestCORSAllowListAndPreflight(t *testing.T) {
	r := corsRouter([]string{"https://app.example.com"})
	if got := do(r, "GET", "https://app.example.com").Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("allowed origin not echoed: %q", got)
	}
	if got := do(r, "GET", "https://evil.example.com").Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("foreign origin must not be allowed: %q", got)
	}
	if rec := do(r, "OPTIONS", "https://app.example.com"); rec.Code != http.StatusNoContent {
		t.Errorf("preflight status %d", rec.Code)
	}
	if rec := do(r, "OPTIONS", "https://evil.example.com"); rec.Code != http.StatusForbidden {
		t.Errorf("foreign preflight status %d", rec.Code)
	}
}

func TestCORSWildcardForDevelopment(t *testing.T) {
	r := corsRouter([]string{"*"})
	if got := do(r, "GET", "http://localhost:8081").Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
		t.Errorf("wildcard should echo origin, got %q", got)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := do(corsRouter(nil), "GET", "").Header()
	if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("X-Frame-Options") != "DENY" {
		t.Errorf("missing hardening headers: %v", h)
	}
}
