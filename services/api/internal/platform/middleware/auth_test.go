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

type stubSessions struct{ active bool }

func (s stubSessions) SessionActive(context.Context, string, string) (bool, error) {
	return s.active, nil
}

var secret = []byte("0123456789abcdef0123456789abcdef")

func token(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func call(t *testing.T, sessions SessionValidator, bearer string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", RequireUser(secret, sessions), func(c *gin.Context) { c.String(http.StatusOK, c.GetString("userID")) })
	req := httptest.NewRequest("GET", "/x", nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Code
}

func TestRequireUserStrictValidation(t *testing.T) {
	future := time.Now().Add(time.Hour).Unix()
	good := jwt.MapClaims{"uid": "u", "sid": "s", "exp": future}

	if got := call(t, stubSessions{true}, token(t, jwt.SigningMethodHS256, secret, good)); got != http.StatusOK {
		t.Fatalf("valid token refused: %d", got)
	}
	cases := map[string]string{
		"missing":       "",
		"garbage":       "abc.def.ghi",
		"wrong secret":  token(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-xx"), good),
		"hs512":         token(t, jwt.SigningMethodHS512, secret, good),
		"no expiry":     token(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "sid": "s"}),
		"expired":       token(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()}),
		"no uid":        token(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"sid": "s", "exp": future}),
		"no session id": token(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"uid": "u", "exp": future}),
		"alg none":      token(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, good),
	}
	for name, tok := range cases {
		if got := call(t, stubSessions{true}, tok); got != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401, got %d", name, got)
		}
	}
	if got := call(t, stubSessions{false}, token(t, jwt.SigningMethodHS256, secret, good)); got != http.StatusUnauthorized {
		t.Fatalf("revoked session must be refused, got %d", got)
	}
}
