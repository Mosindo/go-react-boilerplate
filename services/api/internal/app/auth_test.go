package app_test

import (
	"testing"

	"example.com/api/internal/app"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	email := uniqueEmail("Auth")

	// Email is trimmed and lower-cased; duplicates differing by case conflict.
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + email + "  ", "password": "short"}), 400)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + email + "  ", "password": "correct-horse-1"}), 201)
	var reg authBody
	r.decode(t, &reg)
	if reg.User.Email != lower(email) || reg.User.ProfileComplete {
		t.Fatalf("unexpected user %+v", reg.User)
	}
	if got := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": upper(email), "password": "correct-horse-1"}), 409); got.errCode() != "email_taken" {
		t.Fatalf("expected email_taken, got %s", got.Body)
	}
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": uniqueEmail("long"), "password": string(make([]byte, 80))}), 400)

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": "wrong-password"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "nobody@test.invalid", "password": "wrong-password"}), 401)
	r = e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": "correct-horse-1"}), 200)
	var login authBody
	r.decode(t, &login)

	e.expect(e.do("GET", "/me", "", nil), 401)
	e.expect(e.do("GET", "/me", "garbage", nil), 401)
	e.expect(e.do("GET", "/me", login.AccessToken, nil), 200)

	// Refresh rotates the token: the old refresh token stops working.
	r = e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), 200)
	var refreshed authBody
	r.decode(t, &refreshed)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), 401)
	e.expect(e.do("GET", "/me", refreshed.AccessToken, nil), 200)

	// Logout revokes the session immediately, even for a still-valid access token.
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": refreshed.RefreshToken}), 204)
	e.expect(e.do("GET", "/me", refreshed.AccessToken, nil), 401)
}

func TestTokenValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("tok")
	// A token signed with another secret, or with alg none, is rejected.
	forged := forgeToken(t, "another-secret-another-secret-123456", u.ID)
	e.expect(e.do("GET", "/me", forged, nil), 401)
	e.expect(e.do("GET", "/me", noneToken(u.ID), nil), 401)
}

func TestPasswordResetAndChange(t *testing.T) {
	e := newEnv(t)
	u := e.register("reset")

	// Unknown e-mail: same 202, nothing sent.
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "ghost@test.invalid"}), 202)
	if e.mail.last("ghost@test.invalid") != "" {
		t.Fatal("no mail expected for unknown address")
	}

	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202)
	code := extractCode(t, e.mail.last(u.Email))

	bad := e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": "AAAAAAAAAA", "password": "new-password-1"}), 400)
	if bad.errCode() != "invalid_code" {
		t.Fatalf("expected invalid_code, got %s", bad.Body)
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "password": "new-password-1"}), 204)
	// Single use, and the old session is gone.
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "password": "other-password-1"}), 400)
	e.expect(e.do("GET", "/me", u.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "correct-horse-1"}), 401)
	r := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "new-password-1"}), 200)
	var a authBody
	r.decode(t, &a)

	// Change password: wrong current password refused, others sessions revoked, current one kept.
	other := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "new-password-1"}), 200)
	var o authBody
	other.decode(t, &o)
	e.expect(e.do("POST", "/me/password", a.AccessToken, map[string]string{"currentPassword": "nope-nope-1", "newPassword": "third-password-1"}), 403)
	e.expect(e.do("POST", "/me/password", a.AccessToken, map[string]string{"currentPassword": "new-password-1", "newPassword": "third-password-1"}), 204)
	e.expect(e.do("GET", "/me", a.AccessToken, nil), 200)
	e.expect(e.do("GET", "/me", o.AccessToken, nil), 401)
}

func TestResetCodeAttemptsAreLimited(t *testing.T) {
	e := newEnv(t)
	u := e.register("brute")
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202)
	code := extractCode(t, e.mail.last(u.Email))
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": "BBBBBBBBBB", "password": "new-password-1"}), 400)
	}
	// Even the right code is refused once the attempt budget is spent.
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "password": "new-password-1"}), 400)
}

func TestAuthRateLimit(t *testing.T) {
	e := newEnvWithLimits(t, app.Limits{Auth: 3, Swipe: 100, Message: 100, Report: 100, Upload: 100})
	body := map[string]string{"email": "x@test.invalid", "password": "whatever-1"}
	for i := 0; i < 3; i++ {
		e.expect(e.do("POST", "/auth/login", "", body), 401)
	}
	r := e.expect(e.do("POST", "/auth/login", "", body), 429)
	if r.Header.Get("Retry-After") == "" || r.errCode() != "rate_limited" {
		t.Fatalf("expected Retry-After + rate_limited, got %v %s", r.Header, r.Body)
	}
}
