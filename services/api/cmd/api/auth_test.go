package main

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/middleware"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	r := e.expect(e.do("GET", "/health", nil, ""), http.StatusOK)
	if !strings.Contains(string(r.Body), `"status":"ok"`) {
		t.Fatalf("unexpected body %s", r.Body)
	}
	e.pool.Close()
	e.expect(e.do("GET", "/health", nil, ""), http.StatusServiceUnavailable)
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register("auth")

	e.expect(e.do("POST", "/auth/register", map[string]string{"email": strings.ToUpper(u.Email), "password": testPassword}, ""), http.StatusConflict)
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": uniqueEmail("weak"), "password": "short"}, ""), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": "not-an-email", "password": testPassword}, ""), http.StatusBadRequest)

	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "wrong-password-1"}, ""), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": "nobody@test.invalid", "password": testPassword}, ""), http.StatusUnauthorized)
	login := e.expect(e.do("POST", "/auth/login", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": testPassword}, ""), http.StatusOK)
	var tokens struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	login.decode(t, &tokens)

	me := e.expect(e.do("GET", "/me", nil, tokens.AccessToken), http.StatusOK)
	if !strings.Contains(string(me.Body), u.Email) || strings.Contains(string(me.Body), "password") {
		t.Fatalf("unexpected /me body %s", me.Body)
	}

	refreshed := e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": tokens.RefreshToken}, ""), http.StatusOK)
	var next struct {
		RefreshToken string `json:"refreshToken"`
	}
	refreshed.decode(t, &next)
	if next.RefreshToken == tokens.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": tokens.RefreshToken}, ""), http.StatusUnauthorized)

	e.expect(e.do("POST", "/auth/logout", map[string]string{"refreshToken": next.RefreshToken}, ""), http.StatusNoContent)
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": next.RefreshToken}, ""), http.StatusUnauthorized)
}

func TestProtectedEndpointsRequireAuth(t *testing.T) {
	e := newEnv(t)
	id := "11111111-1111-1111-1111-111111111111"
	for _, c := range [][2]string{
		{"GET", "/me"}, {"DELETE", "/me"}, {"GET", "/profile"}, {"PUT", "/profile"}, {"GET", "/preferences"}, {"GET", "/interests"},
		{"GET", "/profiles/" + id}, {"GET", "/photos"}, {"POST", "/photos"}, {"GET", "/discover"}, {"POST", "/swipes"},
		{"GET", "/conversations"}, {"GET", "/conversations/" + id + "/messages"}, {"POST", "/conversations/" + id + "/messages"},
		{"GET", "/events"}, {"GET", "/notifications"}, {"POST", "/reports"}, {"POST", "/blocks"}, {"GET", "/blocks"},
	} {
		if r := e.do(c[0], c[1], nil, ""); r.Status != http.StatusUnauthorized {
			t.Errorf("%s %s without token: expected 401, got %d", c[0], c[1], r.Status)
		}
	}
	if r := e.do("GET", "/me", nil, "not.a.jwt"); r.Status != http.StatusUnauthorized {
		t.Errorf("garbage token: expected 401, got %d", r.Status)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register("reset")

	e.expect(e.do("POST", "/auth/forgot", map[string]string{"email": "ghost@test.invalid"}, ""), http.StatusAccepted)
	if e.mail.lastBody() != "" {
		t.Fatal("no email may be sent for unknown accounts")
	}

	e.expect(e.do("POST", "/auth/forgot", map[string]string{"email": u.Email}, ""), http.StatusAccepted)
	token := regexp.MustCompile(`\n([A-Za-z0-9_-]{40,})\n`).FindStringSubmatch(e.mail.lastBody())
	if token == nil {
		t.Fatalf("reset token missing from email: %q", e.mail.lastBody())
	}

	e.expect(e.do("POST", "/auth/reset", map[string]string{"token": "bogus", "password": "BrandNewPass99"}, ""), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/reset", map[string]string{"token": token[1], "password": "short"}, ""), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/reset", map[string]string{"token": token[1], "password": "BrandNewPass99"}, ""), http.StatusNoContent)
	e.expect(e.do("POST", "/auth/reset", map[string]string{"token": token[1], "password": "AnotherPass1234"}, ""), http.StatusBadRequest)

	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": testPassword}, ""), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "BrandNewPass99"}, ""), http.StatusOK)
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": u.Refresh}, ""), http.StatusUnauthorized)
}

func TestChangePasswordRevokesSessions(t *testing.T) {
	e := newEnv(t)
	u := e.register("chpw")
	e.expect(e.do("PUT", "/me/password", map[string]string{"currentPassword": "nope-nope-nope", "newPassword": "BrandNewPass99"}, u.Token), http.StatusForbidden)
	e.expect(e.do("PUT", "/me/password", map[string]string{"currentPassword": testPassword, "newPassword": "BrandNewPass99"}, u.Token), http.StatusNoContent)
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": u.Refresh}, ""), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "BrandNewPass99"}, ""), http.StatusOK)
}

func TestRateLimitingOnCredentialEndpoints(t *testing.T) {
	middleware.SetRateLimiting(true)
	defer middleware.SetRateLimiting(false)
	e := newEnv(t)

	var limited bool
	for i := 0; i < 15; i++ {
		r := e.do("POST", "/auth/login", map[string]string{"email": "brute@test.invalid", "password": "guess-guess-guess"}, "")
		if r.Status == http.StatusTooManyRequests {
			limited = true
			if r.Header.Get("Retry-After") == "" {
				t.Fatal("429 must carry Retry-After")
			}
			break
		}
	}
	if !limited {
		t.Fatal("login must be rate limited")
	}
	_ = time.Second
}
