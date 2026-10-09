package main

import (
	"context"
	"net/http"
	"testing"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", nil, ""), http.StatusOK)
	r := e.do("GET", "/health", nil, "")
	if r.Header.Get("X-Request-ID") == "" || r.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing request id / security headers: %v", r.Header)
	}
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	email := uniqueEmail()
	u := e.register("  " + email + " ") // e-mail is trimmed and normalised
	if u.Email == "" {
		t.Fatal("no user")
	}

	// Duplicate (case-insensitive) e-mails are rejected.
	e.expect(e.do("POST", "/auth/register", map[string]any{"email": upper(email), "password": "another good password"}, ""), http.StatusConflict)
	// Invalid input.
	e.expect(e.do("POST", "/auth/register", map[string]any{"email": "nope", "password": "another good password"}, ""), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/register", map[string]any{"email": uniqueEmail(), "password": "short"}, ""), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/register", map[string]any{"email": uniqueEmail()}, ""), http.StatusBadRequest)

	// /me needs a valid token.
	e.expect(e.do("GET", "/me", nil, ""), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", nil, "garbage"), http.StatusUnauthorized)
	me := e.expect(e.do("GET", "/me", nil, u.Token), http.StatusOK).m(t)
	if me["email"] != email {
		t.Fatalf("unexpected email %v", me["email"])
	}
	if onb := me["onboarding"].(map[string]any); onb["complete"] != false {
		t.Fatalf("fresh account must not be onboarded: %v", onb)
	}

	// Login: wrong password / unknown e-mail are indistinguishable.
	bad := e.expect(e.do("POST", "/auth/login", map[string]any{"email": email, "password": "wrong password!"}, ""), http.StatusUnauthorized)
	unknown := e.expect(e.do("POST", "/auth/login", map[string]any{"email": uniqueEmail(), "password": "wrong password!"}, ""), http.StatusUnauthorized)
	if string(bad.Body) != string(unknown.Body) {
		t.Fatalf("login errors leak account existence: %s vs %s", bad.Body, unknown.Body)
	}
	login := e.expect(e.do("POST", "/auth/login", map[string]any{"email": email, "password": "correct horse battery"}, ""), http.StatusOK).m(t)

	// Refresh tokens rotate and are single use.
	rt := login["refreshToken"].(string)
	next := e.expect(e.do("POST", "/auth/refresh", map[string]any{"refreshToken": rt}, ""), http.StatusOK).m(t)
	if next["refreshToken"] == rt {
		t.Fatal("refresh token was not rotated")
	}
	e.expect(e.do("POST", "/auth/refresh", map[string]any{"refreshToken": rt}, ""), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", nil, next["accessToken"].(string)), http.StatusOK)

	// Logout revokes the refresh token.
	e.expect(e.do("POST", "/auth/logout", map[string]any{"refreshToken": next["refreshToken"]}, ""), http.StatusNoContent)
	e.expect(e.do("POST", "/auth/refresh", map[string]any{"refreshToken": next["refreshToken"]}, ""), http.StatusUnauthorized)
}

func upper(s string) string {
	out := []byte(s)
	for i, c := range out {
		if c >= 'a' && c <= 'z' {
			out[i] = c - 32
		}
	}
	return string(out)
}

func TestPasswordChangeRevokesOtherSessions(t *testing.T) {
	e := newEnv(t)
	u := e.register("")
	other := e.expect(e.do("POST", "/auth/login", map[string]any{"email": u.Email, "password": "correct horse battery"}, ""), http.StatusOK).m(t)

	e.expect(e.do("POST", "/me/password", map[string]any{"currentPassword": "nope nope nope", "newPassword": "brand new password"}, u.Token), http.StatusForbidden)
	e.expect(e.do("POST", "/me/password", map[string]any{"currentPassword": "correct horse battery", "newPassword": "short"}, u.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/me/password", map[string]any{"currentPassword": "correct horse battery", "newPassword": "brand new password"}, u.Token), http.StatusNoContent)

	// The session that changed the password survives, the other one is revoked.
	e.expect(e.do("POST", "/auth/refresh", map[string]any{"refreshToken": u.Refresh}, ""), http.StatusOK)
	e.expect(e.do("POST", "/auth/refresh", map[string]any{"refreshToken": other["refreshToken"]}, ""), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", map[string]any{"email": u.Email, "password": "correct horse battery"}, ""), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", map[string]any{"email": u.Email, "password": "brand new password"}, ""), http.StatusOK)
}

func TestPasswordResetFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register("")

	// Same response for known and unknown accounts.
	known := e.expect(e.do("POST", "/auth/password/forgot", map[string]any{"email": u.Email}, ""), http.StatusAccepted)
	unknown := e.expect(e.do("POST", "/auth/password/forgot", map[string]any{"email": uniqueEmail()}, ""), http.StatusAccepted)
	if string(known.Body) != string(unknown.Body) {
		t.Fatal("forgot-password response reveals whether the account exists")
	}
	code := e.mailer.codeFor(t, u.Email)

	// Wrong codes fail and burn attempts; the right code still works afterwards.
	for i := 0; i < 2; i++ {
		r := e.expect(e.do("POST", "/auth/password/reset", map[string]any{"email": u.Email, "code": "AAAAAAAA", "newPassword": "reset password 1"}, ""), http.StatusBadRequest)
		if r.m(t)["code"] != "invalid_code" {
			t.Fatalf("unexpected body %s", r.Body)
		}
	}
	e.expect(e.do("POST", "/auth/password/reset", map[string]any{"email": u.Email, "code": code, "newPassword": "short"}, ""), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/password/reset", map[string]any{"email": u.Email, "code": code, "newPassword": "reset password 1"}, ""), http.StatusNoContent)

	// Single use, sessions revoked, new password works.
	e.expect(e.do("POST", "/auth/password/reset", map[string]any{"email": u.Email, "code": code, "newPassword": "reset password 2"}, ""), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/refresh", map[string]any{"refreshToken": u.Refresh}, ""), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", map[string]any{"email": u.Email, "password": "reset password 1"}, ""), http.StatusOK)
}

func TestPasswordResetLocksAfterTooManyGuesses(t *testing.T) {
	e := newEnv(t)
	u := e.register("")
	e.expect(e.do("POST", "/auth/password/forgot", map[string]any{"email": u.Email}, ""), http.StatusAccepted)
	code := e.mailer.codeFor(t, u.Email)

	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/auth/password/reset", map[string]any{"email": u.Email, "code": "ZZZZZZZZ", "newPassword": "reset password 1"}, ""), http.StatusBadRequest)
	}
	// Even the correct code is now refused: brute forcing 8 characters is impossible.
	e.expect(e.do("POST", "/auth/password/reset", map[string]any{"email": u.Email, "code": code, "newPassword": "reset password 1"}, ""), http.StatusBadRequest)
}

func TestCredentialEndpointsAreRateLimited(t *testing.T) {
	e := newEnv(t)
	limited := false
	for i := 0; i < 30; i++ {
		r := e.do("POST", "/auth/login", map[string]any{"email": uniqueEmail(), "password": "wrong password!"}, "")
		if r.Status == http.StatusTooManyRequests {
			limited = true
			if r.Header.Get("Retry-After") == "" {
				t.Fatal("429 without Retry-After")
			}
			break
		}
	}
	if !limited {
		t.Fatal("expected login to be rate limited")
	}
}

func TestJWTValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("")
	// A token signed with another secret or with alg=none must be refused.
	forged := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOiIxIiwic2lkIjoiMSIsImV4cCI6OTk5OTk5OTk5OX0."
	e.expect(e.do("GET", "/me", nil, forged), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", nil, u.Token+"x"), http.StatusUnauthorized)
}

func TestAccountDeletionRemovesEverything(t *testing.T) {
	e := newEnv(t)
	a, b, convID := e.matchPair()
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", map[string]any{"body": "salut"}, a.Token), http.StatusCreated)
	var key string
	if err := e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}

	e.expect(e.do("DELETE", "/me", map[string]any{"password": "wrong password!!"}, a.Token), http.StatusForbidden)
	e.expect(e.do("DELETE", "/me", map[string]any{"password": "correct horse battery"}, a.Token), http.StatusNoContent)

	// Credentials, data and files are gone.
	e.expect(e.do("POST", "/auth/login", map[string]any{"email": a.Email, "password": "correct horse battery"}, ""), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/refresh", map[string]any{"refreshToken": a.Refresh}, ""), http.StatusUnauthorized)
	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "swipes": "user_id", "messages": "sender_id", "preferences": "user_id", "user_interests": "user_id", "notifications": "user_id"} {
		var n int
		if err := e.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s still has %d rows for the deleted user (err=%v)", table, n, err)
		}
	}
	if _, err := e.store.Open(context.Background(), key); err == nil {
		t.Fatal("photo file still stored after account deletion")
	}
	// The other member no longer sees the match or the conversation.
	if res := e.expect(e.do("GET", "/matches", nil, b.Token), http.StatusOK).m(t); len(res["matches"].([]any)) != 0 {
		t.Fatalf("match survived deletion: %v", res)
	}
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", nil, b.Token), http.StatusNotFound)
}
