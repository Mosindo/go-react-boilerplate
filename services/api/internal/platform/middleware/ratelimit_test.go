package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitBlocksThenRefills(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Unix(1000, 0)
	r := gin.New()
	r.Use(rateLimitWithClock(1, 2, nil, func() time.Time { return now }))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	call := func() int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
		return w.Code
	}
	if call() != 200 || call() != 200 {
		t.Fatal("burst of 2 should pass")
	}
	if call() != http.StatusTooManyRequests {
		t.Fatal("third request should be limited")
	}
	now = now.Add(1500 * time.Millisecond)
	if call() != 200 {
		t.Fatal("bucket should refill after waiting")
	}
}
