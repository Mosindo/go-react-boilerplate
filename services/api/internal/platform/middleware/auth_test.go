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

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func validClaims() AccessClaims {
	now := time.Now()
	return AccessClaims{
		UserID:    "u1",
		SessionID: "s1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
}

func TestParseAccessToken(t *testing.T) {
	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, validClaims())); err != nil {
		t.Fatalf("expected valid token: %v", err)
	}

	noExp := validClaims()
	noExp.ExpiresAt = nil
	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, noExp)); err == nil {
		t.Fatal("expected token without exp to be rejected")
	}

	expired := validClaims()
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, expired)); err == nil {
		t.Fatal("expected expired token to be rejected")
	}

	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS512, testSecret, validClaims())); err == nil {
		t.Fatal("expected HS512 token to be rejected")
	}

	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-xx"), validClaims())); err == nil {
		t.Fatal("expected wrong-secret token to be rejected")
	}

	missing := validClaims()
	missing.SessionID = ""
	if _, err := ParseAccessToken(testSecret, sign(t, jwt.SigningMethodHS256, testSecret, missing)); err == nil {
		t.Fatal("expected token without sid to be rejected")
	}

	unsigned := sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims())
	if _, err := ParseAccessToken(testSecret, unsigned); err == nil {
		t.Fatal("expected alg=none token to be rejected")
	}
}

func TestRequireUserChecksSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	active := true
	r := gin.New()
	r.GET("/p", RequireUser(testSecret, func(_ context.Context, sid, uid string) (bool, error) {
		return active && sid == "s1" && uid == "u1", nil
	}), func(c *gin.Context) { c.String(200, c.GetString("userID")) })

	token := sign(t, jwt.SigningMethodHS256, testSecret, validClaims())
	do := func(header string) int {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := do(""); code != 401 {
		t.Fatalf("missing token: got %d", code)
	}
	if code := do("Bearer " + token); code != 200 {
		t.Fatalf("valid token: got %d", code)
	}
	active = false
	if code := do("Bearer " + token); code != 401 {
		t.Fatalf("revoked session: got %d", code)
	}
}
