package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLimitBodyRejectsOversizedJSONButNotMultipart(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LimitBody(16))
	r.POST("/x", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusNoContent)
	})

	do := func(contentType, body string) int {
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	big := strings.Repeat("a", 100)
	if code := do("application/json", "{}"); code != http.StatusNoContent {
		t.Fatalf("small body: got %d", code)
	}
	if code := do("application/json", big); code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large JSON body must be rejected, got %d", code)
	}
	if code := do("multipart/form-data; boundary=x", big); code != http.StatusNoContent {
		t.Fatalf("multipart is limited by its own handler, got %d", code)
	}
}
