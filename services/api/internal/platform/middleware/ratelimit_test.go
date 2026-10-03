package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimitBlocksAfterBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(0.001, 3, func(*gin.Context) string { return "k" }))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	codes := make([]int, 0, 5)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		codes = append(codes, w.Code)
	}
	want := []int{200, 200, 200, 429, 429}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("request %d: want %d got %d (all=%v)", i, want[i], codes[i], codes)
		}
	}
}
