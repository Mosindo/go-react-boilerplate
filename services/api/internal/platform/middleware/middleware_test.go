package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestLimiterWindow(t *testing.T) {
	now := time.Unix(1000, 0)
	l := newLimiter(3, time.Minute, func() time.Time { return now })
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("call %d should pass", i)
		}
	}
	ok, retry := l.Allow("a")
	if ok || retry <= 0 || retry > time.Minute {
		t.Fatalf("4th call must be limited with a retry hint, got %v %v", ok, retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Error("keys are independent")
	}
	now = now.Add(30 * time.Second)
	if ok, retry := l.Allow("a"); ok || retry > 30*time.Second+time.Second {
		t.Errorf("still inside the window: %v %v", ok, retry)
	}
	now = now.Add(31 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Error("window elapsed: allowed again")
	}
}

func TestLimiterSweepsStaleKeys(t *testing.T) {
	now := time.Unix(1000, 0)
	l := newLimiter(1, time.Minute, func() time.Time { return now })
	for i := 0; i < 100; i++ {
		l.Allow(string(rune('a'+i%26)) + string(rune('A'+i/26)))
	}
	now = now.Add(3 * time.Minute)
	l.Allow("fresh")
	l.mu.Lock()
	n := len(l.buckets)
	l.mu.Unlock()
	if n != 1 {
		t.Errorf("stale buckets should be swept, %d left", n)
	}
}

func TestLimiterConcurrent(t *testing.T) {
	l := NewLimiter(50, time.Minute)
	done := make(chan bool, 200)
	for i := 0; i < 200; i++ {
		go func() { ok, _ := l.Allow("k"); done <- ok }()
	}
	allowed := 0
	for i := 0; i < 200; i++ {
		if <-done {
			allowed++
		}
	}
	if allowed != 50 {
		t.Errorf("exactly 50 of 200 concurrent calls must pass, got %d", allowed)
	}
}

func TestRateLimitMiddlewareResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitByIP(NewLimiter(1, time.Minute)))
	r.GET("/x", func(c *gin.Context) { c.Status(204) })
	do := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/x", nil)
		req.RemoteAddr = "203.0.113.9:1234"
		r.ServeHTTP(w, req)
		return w
	}
	if w := do(); w.Code != 204 {
		t.Fatalf("first: %d", w.Code)
	}
	w := do()
	if w.Code != 429 || w.Header().Get("Retry-After") == "" || w.Body.String() != `{"code":"rate_limited","error":"too many requests, slow down"}` {
		t.Fatalf("limited: %d %v %s", w.Code, w.Header(), w.Body.String())
	}
}

func TestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimit(10))
	r.POST("/x", func(c *gin.Context) {
		buf := make([]byte, 100)
		if _, err := c.Request.Body.Read(buf); err != nil && !errors.Is(err, io.EOF) {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(200)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/x", strings.NewReader("01234567890123456789"))
	r.ServeHTTP(w, req)
	if w.Code != 413 {
		t.Fatalf("declared length above the limit: %d", w.Code)
	}
}

type fakeSessions struct {
	active bool
	err    error
}

func (f fakeSessions) SessionActive(context.Context, string, string) (bool, error) {
	return f.active, f.err
}

var secret = []byte("0123456789abcdef0123456789abcdef")

const (
	uid = "11111111-1111-1111-1111-111111111111"
	sid = "22222222-2222-2222-2222-222222222222"
)

func signed(method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		panic(err)
	}
	return s
}

func callWith(sessions SessionChecker, header string) (int, string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", RequireUser(secret, sessions), func(c *gin.Context) {
		c.String(200, c.GetString("userID")+"|"+c.GetString("sessionID"))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	r.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func TestRequireUser(t *testing.T) {
	exp := time.Now().Add(time.Minute).Unix()
	valid := jwt.MapClaims{"uid": uid, "sid": sid, "exp": exp}
	active := fakeSessions{active: true}

	if code, body := callWith(active, "Bearer "+signed(jwt.SigningMethodHS256, valid, secret)); code != 200 || body != uid+"|"+sid {
		t.Fatalf("valid token: %d %s", code, body)
	}
	if code, _ := callWith(active, "bearer "+signed(jwt.SigningMethodHS256, valid, secret)); code != 200 {
		t.Errorf("scheme is case-insensitive: %d", code)
	}

	rejected := map[string]string{
		"missing header": "",
		"basic scheme":   "Basic abc",
		"alg none":       "Bearer " + signed(jwt.SigningMethodNone, valid, jwt.UnsafeAllowNoneSignatureType),
		"HS512":          "Bearer " + signed(jwt.SigningMethodHS512, valid, secret),
		"wrong key":      "Bearer " + signed(jwt.SigningMethodHS256, valid, []byte("x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x")),
		"expired":        "Bearer " + signed(jwt.SigningMethodHS256, jwt.MapClaims{"uid": uid, "sid": sid, "exp": time.Now().Add(-time.Minute).Unix()}, secret),
		"no expiry":      "Bearer " + signed(jwt.SigningMethodHS256, jwt.MapClaims{"uid": uid, "sid": sid}, secret),
		"no sid":         "Bearer " + signed(jwt.SigningMethodHS256, jwt.MapClaims{"uid": uid, "exp": exp}, secret),
		"no uid":         "Bearer " + signed(jwt.SigningMethodHS256, jwt.MapClaims{"sid": sid, "exp": exp}, secret),
		"non uuid uid":   "Bearer " + signed(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "1 OR 1=1", "sid": sid, "exp": exp}, secret),
		"numeric uid":    "Bearer " + signed(jwt.SigningMethodHS256, jwt.MapClaims{"uid": 7, "sid": sid, "exp": exp}, secret),
		"garbage":        "Bearer abc.def.ghi",
	}
	for name, h := range rejected {
		if code, body := callWith(active, h); code != 401 {
			t.Errorf("%s: want 401, got %d %s", name, code, body)
		}
	}

	// revoked session / deleted user
	if code, _ := callWith(fakeSessions{active: false}, "Bearer "+signed(jwt.SigningMethodHS256, valid, secret)); code != 401 {
		t.Errorf("inactive session must be 401, got %d", code)
	}
	// store failure must not authenticate
	if code, _ := callWith(fakeSessions{err: errors.New("db down")}, "Bearer "+signed(jwt.SigningMethodHS256, valid, secret)); code != 500 {
		t.Errorf("session store error must fail closed with 500, got %d", code)
	}
}
