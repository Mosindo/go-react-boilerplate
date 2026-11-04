package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

type fakeSessions struct{ active bool }

func (f fakeSessions) IsSessionActive(context.Context, string, string) (bool, error) {
	return f.active, nil
}

func token(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func call(t *testing.T, tok string, active bool) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", RequireUser(secret, fakeSessions{active}), func(c *gin.Context) { c.String(200, c.GetString("userID")) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Code
}

func TestRequireUser(t *testing.T) {
	exp := time.Now().Add(time.Minute).Unix()
	valid := jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": exp}

	if got := call(t, token(t, jwt.SigningMethodHS256, secret, valid), true); got != 200 {
		t.Errorf("valid token: %d", got)
	}
	if got := call(t, token(t, jwt.SigningMethodHS256, secret, valid), false); got != 401 {
		t.Errorf("revoked session must be refused: %d", got)
	}
	if got := call(t, "", true); got != 401 {
		t.Errorf("missing token: %d", got)
	}
	if got := call(t, token(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), valid), true); got != 401 {
		t.Errorf("wrong key: %d", got)
	}
	if got := call(t, token(t, jwt.SigningMethodHS512, secret, valid), true); got != 401 {
		t.Errorf("HS512 must be refused (HS256 only): %d", got)
	}
	if got := call(t, token(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid), true); got != 401 {
		t.Errorf("alg none must be refused: %d", got)
	}
	if got := call(t, token(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u1", "sid": "s1"}), true); got != 401 {
		t.Errorf("token without exp must be refused: %d", got)
	}
	if got := call(t, token(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(-time.Minute).Unix()}), true); got != 401 {
		t.Errorf("expired token: %d", got)
	}
	if got := call(t, token(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"sid": "s1", "exp": exp}), true); got != 401 {
		t.Errorf("token without uid: %d", got)
	}
}

func TestLimiter(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := NewLimiter(60, 3)
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("burst request %d refused", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait < 1 {
		t.Fatalf("4th request should be refused with a wait, got ok=%v wait=%d", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be independent")
	}
	now = now.Add(2 * time.Second) // 1 token per second
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("tokens must refill over time")
	}
}

func TestRateLimitMiddlewareReturns429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", RateLimitByIP(NewLimiter(1, 1)), func(c *gin.Context) { c.Status(200) })
	codes := []int{}
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
		if rec.Code == 429 && rec.Header().Get("Retry-After") == "" {
			t.Error("missing Retry-After")
		}
	}
	if codes[0] != 200 || codes[1] != 429 {
		t.Fatalf("codes: %v", codes)
	}
}

func TestValidRequestID(t *testing.T) {
	for _, ok := range []string{"abc-123", "req_1.2", "9f0fc27808b8295f47f5d303"} {
		if !validRequestID(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "a b", "x\r\nSet-Cookie: a=b", `"quoted"`, string(make([]byte, 65))} {
		if validRequestID(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
