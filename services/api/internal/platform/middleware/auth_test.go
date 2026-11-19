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

type fakeSessions struct{ active bool }

func (f fakeSessions) SessionActive(context.Context, string, string) (bool, error) {
	return f.active, nil
}

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func run(t *testing.T, sessions SessionValidator, header string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", RequireUser(testSecret, sessions), func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRequireUser(t *testing.T) {
	valid := jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix()}

	if code := run(t, fakeSessions{true}, "Bearer "+sign(t, jwt.SigningMethodHS256, testSecret, valid)); code != http.StatusOK {
		t.Fatalf("valid token: got %d", code)
	}
	if code := run(t, fakeSessions{false}, "Bearer "+sign(t, jwt.SigningMethodHS256, testSecret, valid)); code != http.StatusUnauthorized {
		t.Fatalf("revoked session: got %d", code)
	}
	if code := run(t, fakeSessions{true}, ""); code != http.StatusUnauthorized {
		t.Fatalf("missing token: got %d", code)
	}

	noExp := jwt.MapClaims{"uid": "u1", "sid": "s1"}
	if code := run(t, fakeSessions{true}, "Bearer "+sign(t, jwt.SigningMethodHS256, testSecret, noExp)); code != http.StatusUnauthorized {
		t.Fatalf("token without exp: got %d", code)
	}

	expired := jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(-time.Minute).Unix()}
	if code := run(t, fakeSessions{true}, "Bearer "+sign(t, jwt.SigningMethodHS256, testSecret, expired)); code != http.StatusUnauthorized {
		t.Fatalf("expired token: got %d", code)
	}

	if code := run(t, fakeSessions{true}, "Bearer "+sign(t, jwt.SigningMethodHS512, testSecret, valid)); code != http.StatusUnauthorized {
		t.Fatalf("HS512 token: got %d", code)
	}

	if code := run(t, fakeSessions{true}, "Bearer "+sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), valid)); code != http.StatusUnauthorized {
		t.Fatalf("wrong secret: got %d", code)
	}

	none := sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid)
	if code := run(t, fakeSessions{true}, "Bearer "+none); code != http.StatusUnauthorized {
		t.Fatalf("alg none: got %d", code)
	}

	missingSid := jwt.MapClaims{"uid": "u1", "exp": time.Now().Add(time.Minute).Unix()}
	if code := run(t, fakeSessions{true}, "Bearer "+sign(t, jwt.SigningMethodHS256, testSecret, missingSid)); code != http.StatusUnauthorized {
		t.Fatalf("missing sid: got %d", code)
	}
}
