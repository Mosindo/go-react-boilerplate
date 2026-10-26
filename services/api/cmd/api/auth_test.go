package main

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/middleware"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	r := e.expect(e.do("GET", "/health", "", nil), 200, "health")
	if r.str("status") != "ok" {
		t.Fatalf("unexpected health body %s", r.Raw)
	}
}

func TestAuthRegisterLoginMeRefreshLogout(t *testing.T) {
	e := newEnv(t)
	email := uniqueEmail("auth")

	// registration normalises email and enforces password rules
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "short"}), 400, "short password")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "onlyletters"}), 400, "no digit")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": testPassword}), 400, "bad email")
	reg := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": testPassword}), 201, "register")
	if reg.str("user", "email") != email {
		t.Fatalf("email must be trimmed and lowercased, got %q", reg.str("user", "email"))
	}
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": testPassword}), 409, "duplicate")

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": "Wrong12345"}), 401, "wrong password")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "nobody@test.example", "password": testPassword}), 401, "unknown user")
	login := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": testPassword}), 200, "login")

	me := e.expect(e.do("GET", "/me", login.str("accessToken"), nil), 200, "me")
	if me.str("email") != email {
		t.Fatalf("me email mismatch: %s", me.Raw)
	}
	e.expect(e.do("GET", "/me", "", nil), 401, "no token")
	e.expect(e.do("GET", "/me", "garbage", nil), 401, "bad token")

	// refresh rotates the refresh token; the old one dies
	rt := login.str("refreshToken")
	ref := e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": rt}), 200, "refresh")
	if ref.str("refreshToken") == rt {
		t.Fatal("refresh token must rotate")
	}
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": rt}), 401, "reuse of rotated token")
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": ref.str("refreshToken")}), 204, "logout")
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": ref.str("refreshToken")}), 401, "refresh after logout")
}

func TestAuthRejectsForgedAndExpiredTokens(t *testing.T) {
	e := newEnv(t)
	u := e.register(uniqueEmail("jwt"))

	// token signed with another secret
	forged := signTestToken(t, "another-secret-another-secret-another", u.ID, time.Now().Add(time.Hour))
	e.expect(e.do("GET", "/me", forged, nil), 401, "forged signature")
	// expired
	expired := signTestToken(t, testSecret, u.ID, time.Now().Add(-time.Minute))
	e.expect(e.do("GET", "/me", expired, nil), 401, "expired token")
	// no expiry
	noExp := signTestToken(t, testSecret, u.ID, time.Time{})
	e.expect(e.do("GET", "/me", noExp, nil), 401, "token without exp")
	// valid one still works
	valid := signTestToken(t, testSecret, u.ID, time.Now().Add(time.Hour))
	e.expect(e.do("GET", "/me", valid, nil), 200, "valid token")

	if _, _, err := middleware.ParseAccessToken([]byte(testSecret), "a.b.c"); err == nil {
		t.Fatal("garbage must not parse")
	}
}

func TestPasswordRecoveryFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register(uniqueEmail("reset"))

	// unknown email gets the same answer and sends nothing
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "ghost@test.example"}), 202, "forgot unknown")
	if to, _ := e.mail.last(); to != "" {
		t.Fatal("no email must be sent for unknown accounts")
	}

	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202, "forgot")
	to, body := e.mail.last()
	if to != u.Email {
		t.Fatalf("mail sent to %q", to)
	}
	code := regexp.MustCompile(`\d{8}`).FindString(body)
	if code == "" {
		t.Fatalf("mail should contain an 8 digit code: %q", body)
	}

	bad := "00000000"
	if code == bad {
		bad = "11111111"
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": bad, "newPassword": "NewPassword456"}), 400, "wrong code")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "weak"}), 400, "weak new password")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}), 204, "reset")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Another789"}), 400, "code is single use")

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": testPassword}), 401, "old password")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword456"}), 200, "new password")
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), 401, "old sessions revoked")
}

func TestPasswordResetLocksAfterTooManyWrongCodes(t *testing.T) {
	e := newEnv(t)
	u := e.register(uniqueEmail("lock"))
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202, "forgot")
	_, body := e.mail.last()
	code := regexp.MustCompile(`\d{8}`).FindString(body)
	wrong := "12345678"
	if code == wrong {
		wrong = "87654321"
	}
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": wrong, "newPassword": "NewPassword456"}), 400, "wrong guess")
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}), 400, "correct code after lock-out")
}

func TestChangePasswordKeepsCurrentSessionOnly(t *testing.T) {
	e := newEnv(t)
	u := e.register(uniqueEmail("chg"))
	other := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": testPassword}), 200, "second session")

	e.expect(e.do("POST", "/me/password", u.Token, map[string]string{"currentPassword": "Wrong12345", "newPassword": "NewPassword456"}), 403, "wrong current")
	e.expect(e.do("POST", "/me/password", u.Token, map[string]string{"currentPassword": testPassword, "newPassword": "NewPassword456"}), 204, "change")
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": other.str("refreshToken")}), 401, "other session revoked")
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), 200, "current session kept")
}

func TestCredentialRateLimit(t *testing.T) {
	e := newEnv(t, func(l *limits) { l.credentials = middleware.NewLimiter(3, time.Minute) })
	for i := 0; i < 3; i++ {
		e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "x@test.example", "password": "Wrong12345"}), 401, "login attempt")
	}
	r := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "x@test.example", "password": "Wrong12345"}), 429, "rate limited")
	if r.Status != 429 {
		t.Fatal("expected 429")
	}
}

func TestForgotPasswordCooldownAndBodyLimit(t *testing.T) {
	e := newEnv(t)
	u := e.register(uniqueEmail("cool"))
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202, "first request")
	_, first := e.mail.last()
	e.mail.mu.Lock()
	before := len(e.mail.sent)
	e.mail.mu.Unlock()
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202, "second request is accepted silently")
	e.mail.mu.Lock()
	after := len(e.mail.sent)
	e.mail.mu.Unlock()
	if after != before || first == "" {
		t.Fatalf("a second code must not be mailed within the cooldown (%d -> %d)", before, after)
	}

	huge := map[string]string{"email": u.Email, "password": strings.Repeat("a", 2<<20)}
	e.expect(e.do("POST", "/auth/login", "", huge), 400, "oversized JSON body")
}
