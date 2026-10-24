package main

import (
	"context"
	"net/http"
	"regexp"
	"testing"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), http.StatusOK)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	email := uniqueEmail("auth")

	// validation
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "nope", "password": "Passw0rd!x"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "short"}), http.StatusBadRequest)

	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + email + "  ", "password": "Passw0rd!x"}), http.StatusCreated)
	userID := r.str(t, "user", "id")
	t.Cleanup(func() { _, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })
	if r.str(t, "user", "email") != email {
		t.Fatalf("email must be normalized, got %s", r.str(t, "user", "email"))
	}
	if contains(r.Body, "password") {
		t.Fatal("response must never contain password material")
	}

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Passw0rd!x"}), http.StatusConflict)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": "wrong-password"}), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "ghost@test.invalid", "password": "wrong-password"}), http.StatusUnauthorized)

	login := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": "Passw0rd!x"}), http.StatusOK)
	access, refresh := login.str(t, "accessToken"), login.str(t, "refreshToken")

	me := e.expect(e.do("GET", "/me", access, nil), http.StatusOK)
	if me.str(t, "id") != userID {
		t.Fatalf("unexpected /me: %s", me.Body)
	}
	e.expect(e.do("GET", "/me", "", nil), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", "garbage.token.value", nil), http.StatusUnauthorized)

	// refresh rotates: the old refresh token stops working
	rotated := e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh}), http.StatusOK)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh}), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": rotated.str(t, "refreshToken")}), http.StatusNoContent)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": rotated.str(t, "refreshToken")}), http.StatusUnauthorized)
}

func TestPasswordRecovery(t *testing.T) {
	e := newEnv(t)
	p := e.register("recover")

	// unknown emails get the same answer (no account enumeration)
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": "nobody@test.invalid"}), http.StatusNoContent)
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": p.Email}), http.StatusNoContent)

	code := regexp.MustCompile(`: ([A-Z2-9]{8})`).FindStringSubmatch(e.mail.body(p.Email))
	if code == nil {
		t.Fatalf("no reset code mailed: %q", e.mail.body(p.Email))
	}

	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": p.Email, "code": "AAAAAAAA", "newPassword": "NewPassw0rd!"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": p.Email, "code": code[1], "newPassword": "short"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": p.Email, "code": code[1], "newPassword": "NewPassw0rd!"}), http.StatusNoContent)
	// a code is single use
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": p.Email, "code": code[1], "newPassword": "Another0ne!"}), http.StatusBadRequest)

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": p.Email, "password": "Passw0rd!x"}), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": p.Email, "password": "NewPassw0rd!"}), http.StatusOK)
}

func TestResetCodeLocksAfterFiveWrongAttempts(t *testing.T) {
	e := newEnv(t)
	p := e.register("lock")
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": p.Email}), http.StatusNoContent)
	code := regexp.MustCompile(`: ([A-Z2-9]{8})`).FindStringSubmatch(e.mail.body(p.Email))[1]
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": p.Email, "code": "BBBBBBBB", "newPassword": "NewPassw0rd!"}), http.StatusBadRequest)
	}
	// even the right code is refused once attempts are exhausted
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": p.Email, "code": code, "newPassword": "NewPassw0rd!"}), http.StatusBadRequest)
}

func TestSuspendedAccountCannotLogin(t *testing.T) {
	e := newEnv(t)
	p := e.register("suspended")
	if _, err := e.pool.Exec(context.Background(), `UPDATE users SET status = 'suspended' WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": p.Email, "password": "Passw0rd!x"}), http.StatusForbidden)
}

func TestLoginIsRateLimited(t *testing.T) {
	e := newEnvScaled(t, 1)
	limited := false
	for i := 0; i < 30; i++ {
		r := e.do("POST", "/auth/login", "", map[string]string{"email": "brute@test.invalid", "password": "guess-guess"})
		if r.Status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("expected brute-force login attempts to be rate limited")
	}
}
