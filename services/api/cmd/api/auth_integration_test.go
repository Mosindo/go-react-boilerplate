package main

import (
	"example.com/api/internal/platform/db"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthRegisterLoginMe(t *testing.T) {
	e := newEnv(t)
	email := e.newEmail("auth")

	r := e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": email, "password": "Password123"})
	e.mustStatus(r, http.StatusCreated, "register")
	if r.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("token responses must not be cacheable")
	}
	body := r.json(t)
	token := body["accessToken"].(string)

	dup := e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": email, "password": "Password123"})
	e.mustStatus(dup, http.StatusConflict, "duplicate register")
	if dup.code(t) != "email_exists" {
		t.Fatalf("unexpected error code %q", dup.code(t))
	}

	bad := e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": email, "password": "wrongpassword"})
	e.mustStatus(bad, http.StatusUnauthorized, "bad login")
	if bad.code(t) != "invalid_credentials" {
		t.Fatalf("unexpected error code %q", bad.code(t))
	}

	e.mustStatus(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Password123"}), http.StatusBadRequest, "invalid email")
	e.mustStatus(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": e.newEmail("short"), "password": "short"}), http.StatusBadRequest, "short password")

	login := e.mustStatus(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": "  " + email + "  ", "password": "Password123"}), http.StatusOK, "login")
	if login.json(t)["accessToken"] == "" {
		t.Fatal("login must return an access token")
	}

	me := e.mustStatus(e.do(http.MethodGet, "/me", token, nil), http.StatusOK, "me").json(t)
	if me["email"] != email {
		t.Fatalf("unexpected /me payload: %v", me)
	}
	if _, leaked := me["passwordHash"]; leaked {
		t.Fatal("/me must never expose the password hash")
	}
}

func TestProtectedRoutesRejectBadTokens(t *testing.T) {
	e := newEnv(t)
	p := e.register("tokens")

	paths := []string{"/me", "/me/profile", "/discover", "/chats", "/notifications", "/photos", "/blocks"}
	for _, path := range paths {
		e.mustStatus(e.do(http.MethodGet, path, "", nil), http.StatusUnauthorized, "no token "+path)
		e.mustStatus(e.do(http.MethodGet, path, "garbage", nil), http.StatusUnauthorized, "garbage token "+path)
	}

	claims := func(method jwt.SigningMethod, secret any, exp time.Time) string {
		token := jwt.NewWithClaims(method, jwt.MapClaims{"uid": p.ID, "sid": "00000000-0000-0000-0000-000000000000", "exp": exp.Unix()})
		signed, err := token.SignedString(secret)
		if err != nil {
			t.Fatal(err)
		}
		return signed
	}
	e.mustStatus(e.do(http.MethodGet, "/me", claims(jwt.SigningMethodHS256, []byte(testSecret), time.Now().Add(-time.Minute)), nil), http.StatusUnauthorized, "expired token")
	e.mustStatus(e.do(http.MethodGet, "/me", claims(jwt.SigningMethodHS256, []byte("another-secret-another-secret-1234"), time.Now().Add(time.Hour)), nil), http.StatusUnauthorized, "wrong secret")
	e.mustStatus(e.do(http.MethodGet, "/me", claims(jwt.SigningMethodHS512, []byte(testSecret), time.Now().Add(time.Hour)), nil), http.StatusUnauthorized, "wrong algorithm")
	// Correctly signed but pointing at a session that does not exist.
	e.mustStatus(e.do(http.MethodGet, "/me", claims(jwt.SigningMethodHS256, []byte(testSecret), time.Now().Add(time.Hour)), nil), http.StatusUnauthorized, "unknown session")

	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"uid": p.ID, "sid": "x", "exp": time.Now().Add(time.Hour).Unix()})
	signed, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	e.mustStatus(e.do(http.MethodGet, "/me", signed, nil), http.StatusUnauthorized, "alg none")
}

func TestLogoutInvalidatesAccessTokenImmediately(t *testing.T) {
	e := newEnv(t)
	p := e.register("logout")
	e.mustStatus(e.do(http.MethodGet, "/me", p.Token, nil), http.StatusOK, "me before logout")
	e.mustStatus(e.do(http.MethodPost, "/auth/logout", "", map[string]string{"refreshToken": p.Refresh}), http.StatusNoContent, "logout")
	r := e.mustStatus(e.do(http.MethodGet, "/me", p.Token, nil), http.StatusUnauthorized, "me after logout")
	if r.code(t) != "session_expired" {
		t.Fatalf("unexpected code %q", r.code(t))
	}
}

func TestPasswordRecoveryOverHTTP(t *testing.T) {
	e := newEnv(t)
	p := e.register("recover")

	e.mustStatus(e.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": "nobody-" + p.Email}), http.StatusAccepted, "forgot unknown")
	e.mustStatus(e.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": p.Email}), http.StatusAccepted, "forgot")
	code := e.mail.code(t)

	bad := e.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"email": p.Email, "code": "abcdef", "newPassword": "BrandNewPass1"})
	e.mustStatus(bad, http.StatusBadRequest, "wrong code")
	if bad.code(t) != "invalid_reset_code" {
		t.Fatalf("unexpected code %q", bad.code(t))
	}
	e.mustStatus(e.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"email": p.Email, "code": code, "newPassword": "BrandNewPass1"}), http.StatusNoContent, "reset")
	e.mustStatus(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": p.Email, "password": "BrandNewPass1"}), http.StatusOK, "login with new password")
	e.mustStatus(e.do(http.MethodGet, "/me", p.Token, nil), http.StatusUnauthorized, "old session after reset")
}

func TestPurgeExpiredRemovesOnlyDeadRows(t *testing.T) {
	e := newEnv(t)
	p := e.register("purge")
	ctx := contextBackground()

	// One long-dead session and one live session for the same user.
	if _, err := e.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, 'dead-hash-purge-test', NOW() - INTERVAL '30 days')
	`, p.ID); err != nil {
		t.Fatal(err)
	}
	removed, err := db.PurgeExpired(ctx, e.pool)
	if err != nil {
		t.Fatal(err)
	}
	if removed < 1 {
		t.Fatalf("expected the expired session to be purged, removed=%d", removed)
	}
	var dead, live int
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessions WHERE token_hash = 'dead-hash-purge-test'`).Scan(&dead)
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessions WHERE user_id = $1`, p.ID).Scan(&live)
	if dead != 0 || live != 1 {
		t.Fatalf("expected only the live session to remain, dead=%d live=%d", dead, live)
	}
	e.mustStatus(e.do(http.MethodGet, "/me", p.Token, nil), http.StatusOK, "live session still works")
}
