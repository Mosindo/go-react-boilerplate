package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	l := NewLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d should pass within the burst", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 || wait > 21*time.Second {
		t.Fatalf("4th request must be refused with a ~20s wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys are independent")
	}
	now = now.Add(21 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("a token must have refilled after 21s")
	}
}

func TestLimiterSweepsIdleBuckets(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	l := NewLimiter(2, time.Minute)
	l.now = func() time.Time { return now }
	for _, k := range []string{"a", "b", "c"} {
		l.Allow(k)
	}
	now = now.Add(10 * time.Minute)
	l.Allow("d")
	if len(l.buckets) != 1 {
		t.Fatalf("idle buckets must be swept, have %d", len(l.buckets))
	}
}

func TestRateLimitMiddlewareReturns429WithRetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitByIP(NewLimiter(2, time.Minute)))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	codes := make([]int, 0, 3)
	var retry string
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		codes = append(codes, rec.Code)
		retry = rec.Header().Get("Retry-After")
	}
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 429 || retry == "" {
		t.Fatalf("unexpected codes %v retry=%q", codes, retry)
	}
}

func TestNilLimiterIsANoop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitByUser(nil))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 200 {
		t.Fatalf("got %d", rec.Code)
	}
}
