package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLimiterRefillsOverTime(t *testing.T) {
	l := NewLimiter(60) // 1/sec, burst 60
	clock := time.Unix(1000, 0)
	l.now = func() time.Time { return clock }

	for i := 0; i < 60; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d should pass", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 {
		t.Fatalf("expected refusal with wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("other keys must have their own budget")
	}
	clock = clock.Add(2 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("bucket should have refilled")
	}
}

func TestRateLimitByIPReturns429WithRetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitByIP(NewLimiter(2), "t"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	codes := []int{}
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		r.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
		if rec.Code == http.StatusTooManyRequests && rec.Header().Get("Retry-After") == "" {
			t.Fatal("missing Retry-After")
		}
	}
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 429 {
		t.Fatalf("unexpected codes %v", codes)
	}
}
