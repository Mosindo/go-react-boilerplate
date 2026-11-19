package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsUUID(t *testing.T) {
	if !IsUUID("2f0b8d5e-6a64-4a1e-9c3f-0d2b5f6a7c11") {
		t.Fatal("expected valid uuid")
	}
	for _, v := range []string{"", "abc", "2f0b8d5e6a644a1e9c3f0d2b5f6a7c11", "' OR 1=1 --"} {
		if IsUUID(v) {
			t.Fatalf("expected %q to be rejected", v)
		}
	}
}

func TestLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		query string
		want  int
		ok    bool
	}{
		{"", 20, true},
		{"limit=5", 5, true},
		{"limit=0", 0, false},
		{"limit=101", 0, false},
		{"limit=abc", 0, false},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+tc.query, nil)
		got, ok := Limit(c, 20, 100)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("query %q: got (%d,%v) want (%d,%v)", tc.query, got, ok, tc.want, tc.ok)
		}
		if !ok && w.Code != http.StatusBadRequest {
			t.Fatalf("query %q: expected 400, got %d", tc.query, w.Code)
		}
	}
}
