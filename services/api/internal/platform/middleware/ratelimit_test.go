package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLimiterWindow(t *testing.T) {
	l := &limiter{hits: map[string]*window{}, limit: 2, period: time.Minute}
	now := time.Now()
	for i := 0; i < 2; i++ {
		if ok, _ := l.allow("a", now); !ok {
			t.Fatalf("request %d should pass", i)
		}
	}
	if ok, wait := l.allow("a", now); ok || wait <= 0 {
		t.Fatal("third request must be rejected")
	}
	if ok, _ := l.allow("b", now); !ok {
		t.Fatal("other key unaffected")
	}
	if ok, _ := l.allow("a", now.Add(time.Minute+time.Second)); !ok {
		t.Fatal("window must reset")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", RateLimit(1, time.Minute, ByIP), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for i, want := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != want {
			t.Fatalf("call %d: got %d want %d", i, w.Code, want)
		}
	}
}
