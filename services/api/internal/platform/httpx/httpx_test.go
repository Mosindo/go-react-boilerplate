package httpx

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCursorRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	at := time.Date(2026, 5, 4, 3, 2, 1, 123456000, time.UTC)
	id := "7d8f3c1e-1111-4222-8333-444455556666"
	token := EncodeCursor(at, id)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/?cursor="+token, nil)
	cur, ok := DecodeCursor(c)
	if !ok || cur == nil || !cur.At.Equal(at) || cur.ID != id {
		t.Fatalf("unexpected cursor %+v", cur)
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/?cursor=bm90LWEtY3Vyc29y", nil)
	if _, ok := DecodeCursor(c); ok || rec.Code != 400 {
		t.Fatal("garbage cursor must be rejected with 400")
	}
}

func TestLimitIsStrict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for raw, wantOK := range map[string]bool{"": true, "5": true, "0": false, "-1": false, "abc": false, "101": false} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("GET", "/?limit="+raw, nil)
		if _, ok := Limit(c, 20, 100); ok != wantOK {
			t.Fatalf("limit %q: ok=%v want %v", raw, ok, wantOK)
		}
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("7d8f3c1e-1111-4222-8333-444455556666") || IsUUID("7d8f3c1e11114222833344445555666600") || IsUUID("zzzzzzzz-1111-4222-8333-444455556666") {
		t.Fatal("unexpected IsUUID result")
	}
}
