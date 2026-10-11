package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("k3v9Xq2LmZp8Rt5Wy7Ub4NcAe1HdGfJs")

func protected(secret []byte) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", middleware.RequireUser(secret), func(c *gin.Context) { c.String(http.StatusOK, c.GetString("userID")) })
	return r
}

func call(r *gin.Engine, token string) int {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestAccessTokenAccepted(t *testing.T) {
	tok, err := signAccessToken(secret, "user-1", "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if code := call(protected(secret), tok); code != http.StatusOK {
		t.Fatalf("valid token rejected: %d", code)
	}
}

func TestAccessTokenRejections(t *testing.T) {
	r := protected(secret)
	mk := func(method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	future := time.Now().Add(time.Hour).Unix()
	good := jwt.MapClaims{"uid": "u", "sid": "s", "iss": middleware.TokenIssuer, "exp": future}

	cases := map[string]string{
		"none":         "",
		"garbage":      "abc.def.ghi",
		"wrong key":    mk(jwt.SigningMethodHS256, good, []byte("another-secret-another-secret-0000")),
		"hs512":        mk(jwt.SigningMethodHS512, good, secret),
		"expired":      mk(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s", "iss": middleware.TokenIssuer, "exp": time.Now().Add(-time.Minute).Unix()}, secret),
		"no exp":       mk(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s", "iss": middleware.TokenIssuer}, secret),
		"wrong issuer": mk(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "sid": "s", "iss": "other", "exp": future}, secret),
		"missing uid":  mk(jwt.SigningMethodHS256, jwt.MapClaims{"sid": "s", "iss": middleware.TokenIssuer, "exp": future}, secret),
		"missing sid":  mk(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u", "iss": middleware.TokenIssuer, "exp": future}, secret),
		"alg none":     "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOiJ1In0.",
	}
	for name, tok := range cases {
		if code := call(r, tok); code != http.StatusUnauthorized {
			t.Errorf("%s: got %d want 401", name, code)
		}
	}
}

func TestHelpers(t *testing.T) {
	if normalizeEmail("  Foo@Bar.COM ") != "foo@bar.com" {
		t.Fatal("email not normalised")
	}
	if hashToken("") != "" || hashToken("a") == hashToken("b") || len(hashToken("a")) != 64 {
		t.Fatal("hashToken misbehaves")
	}
	if validPassword("short") || validPassword(string(make([]byte, 73))) || !validPassword("long enough pw") {
		t.Fatal("password length rules broken")
	}
}
