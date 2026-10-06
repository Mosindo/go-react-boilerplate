package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	r := e.expect(e.do(http.MethodGet, "/health", "", nil), http.StatusOK)
	if !strings.Contains(string(r.Body), `"status":"ok"`) {
		t.Fatalf("unexpected body %s", r.Body)
	}
}

func TestAuthRegisterLoginMe(t *testing.T) {
	e := newEnv(t)
	u := e.register("Alice")

	// Email is trimmed and lowercased; login works with any casing.
	login := e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email": "  " + strings.ToUpper(u.Email) + " ", "password": "Password123!",
	}), http.StatusOK)
	if login.json(t)["accessToken"] == "" {
		t.Fatal("login must return an access token")
	}

	me := e.expect(e.do(http.MethodGet, "/me", u.Token, nil), http.StatusOK).json(t)
	if me["id"] != u.ID || me["email"] != u.Email {
		t.Fatalf("unexpected /me %v", me)
	}
	if _, leaked := me["passwordHash"]; leaked {
		t.Fatal("password hash must never be exposed")
	}

	// The password is stored hashed, never in clear.
	var hash string
	if err := e.pool.QueryRow(t.Context(), `SELECT password_hash FROM users WHERE id=$1`, u.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$2") || strings.Contains(hash, "Password123!") {
		t.Fatalf("password must be bcrypt-hashed, got %q", hash)
	}
}

func TestAuthRejectsBadInput(t *testing.T) {
	e := newEnv(t)
	u := e.register("bob")

	cases := []struct {
		name string
		path string
		body map[string]string
		want int
	}{
		{"duplicate email", "/auth/register", map[string]string{"email": u.Email, "password": "Password123!"}, http.StatusConflict},
		{"invalid email", "/auth/register", map[string]string{"email": "nope", "password": "Password123!"}, http.StatusBadRequest},
		{"short password", "/auth/register", map[string]string{"email": "x@test.local", "password": "short"}, http.StatusBadRequest},
		{"password over bcrypt limit", "/auth/register", map[string]string{"email": "y@test.local", "password": strings.Repeat("a", 100)}, http.StatusBadRequest},
		{"wrong password", "/auth/login", map[string]string{"email": u.Email, "password": "WrongPassword1"}, http.StatusUnauthorized},
		{"unknown email", "/auth/login", map[string]string{"email": "ghost@test.local", "password": "Password123!"}, http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e.expect(e.do(http.MethodPost, c.path, "", c.body), c.want)
		})
	}
}

func TestAuthRefreshRotationAndLogout(t *testing.T) {
	e := newEnv(t)
	u := e.register("carol")

	r := e.expect(e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), http.StatusOK).json(t)
	newRefresh, _ := r["refreshToken"].(string)
	newAccess, _ := r["accessToken"].(string)
	if newRefresh == "" || newRefresh == u.Refresh {
		t.Fatal("refresh token must rotate")
	}
	// The consumed refresh token is dead.
	e.expect(e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), http.StatusUnauthorized)

	e.expect(e.do(http.MethodGet, "/me", newAccess, nil), http.StatusOK)
	e.expect(e.do(http.MethodPost, "/auth/logout", "", map[string]string{"refreshToken": newRefresh}), http.StatusNoContent)
	// Logout takes effect immediately, not when the 15-minute access token expires.
	e.expect(e.do(http.MethodGet, "/me", newAccess, nil), http.StatusUnauthorized)
}

func TestAuthTokenValidationIsStrict(t *testing.T) {
	e := newEnv(t)
	u := e.register("dave")

	sign := func(method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	valid := jwt.MapClaims{"uid": u.ID, "oid": "x", "sid": "00000000-0000-0000-0000-000000000000", "exp": time.Now().Add(time.Hour).Unix()}

	e.expect(e.do(http.MethodGet, "/me", "", nil), http.StatusUnauthorized)
	e.expect(e.do(http.MethodGet, "/me", "garbage", nil), http.StatusUnauthorized)
	// alg=none
	e.expect(e.do(http.MethodGet, "/me", sign(jwt.SigningMethodNone, valid, jwt.UnsafeAllowNoneSignatureType), nil), http.StatusUnauthorized)
	// Wrong secret.
	e.expect(e.do(http.MethodGet, "/me", sign(jwt.SigningMethodHS256, valid, []byte("another-secret-another-secret-12345")), nil), http.StatusUnauthorized)
	// Other HMAC algorithm.
	e.expect(e.do(http.MethodGet, "/me", sign(jwt.SigningMethodHS512, valid, []byte(testJWTSecret)), nil), http.StatusUnauthorized)
	// Expired.
	expired := jwt.MapClaims{"uid": u.ID, "oid": "x", "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()}
	e.expect(e.do(http.MethodGet, "/me", sign(jwt.SigningMethodHS256, expired, []byte(testJWTSecret)), nil), http.StatusUnauthorized)
	// Missing expiration.
	noExp := jwt.MapClaims{"uid": u.ID, "oid": "x", "sid": "s"}
	e.expect(e.do(http.MethodGet, "/me", sign(jwt.SigningMethodHS256, noExp, []byte(testJWTSecret)), nil), http.StatusUnauthorized)
	// Correctly signed but session unknown to the database.
	e.expect(e.do(http.MethodGet, "/me", sign(jwt.SigningMethodHS256, valid, []byte(testJWTSecret)), nil), http.StatusUnauthorized)
}

func TestAuthLoginIsRateLimitedPerIP(t *testing.T) {
	e := newEnv(t)
	body := []byte(`{"email":"nobody@test.local","password":"Whatever123"}`)
	var last response
	for i := 0; i < 25; i++ {
		last = e.raw(http.MethodPost, "/auth/login", "", "203.0.113.50", body, "application/json")
	}
	if last.Status != http.StatusTooManyRequests || last.Header.Get("Retry-After") == "" {
		t.Fatalf("expected 429 with Retry-After, got %d", last.Status)
	}
	// Another client is unaffected.
	e.expect(e.raw(http.MethodPost, "/auth/login", "", "203.0.113.51", body, "application/json"), http.StatusUnauthorized)
}

func TestPasswordRecoveryFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register("erin")

	// Unknown emails get the same answer (no account enumeration).
	e.expect(e.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": "ghost@test.local"}), http.StatusAccepted)
	e.expect(e.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": u.Email}), http.StatusAccepted)
	code := e.mail.codeFor(u.Email)
	if code == "" {
		t.Fatal("a reset code must be emailed")
	}

	e.expect(e.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": "WRONGCODE", "newPassword": "BrandNew123!"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": strings.ToLower(code), "newPassword": "BrandNew123!"}), http.StatusNoContent)

	// The code is single-use, old password dead, new password works, old sessions revoked.
	e.expect(e.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Another123!!"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123!"}), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "BrandNew123!"}), http.StatusOK)
	e.expect(e.do(http.MethodGet, "/me", u.Token, nil), http.StatusUnauthorized)
}

func TestPasswordRecoveryLocksAfterTooManyWrongCodes(t *testing.T) {
	e := newEnv(t)
	u := e.register("frank")
	e.expect(e.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": u.Email}), http.StatusAccepted)
	code := e.mail.codeFor(u.Email)

	for i := 0; i < 5; i++ {
		e.expect(e.raw(http.MethodPost, "/auth/password/reset", "", "198.51.100."+string(rune('1'+i)),
			[]byte(`{"email":"`+u.Email+`","code":"AAAAAAAA","newPassword":"BrandNew123!"}`), "application/json"), http.StatusBadRequest)
	}
	// Even the right code no longer works: the attempt budget is spent.
	e.expect(e.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "BrandNew123!"}), http.StatusBadRequest)
}
