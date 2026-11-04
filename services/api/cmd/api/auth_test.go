package main

import (
	"net/http"
	"testing"
)

func TestHealth(t *testing.T) {
	res := do(t, newRouter(), http.MethodGet, "/health", nil, "")
	if res.Status != 200 {
		t.Fatalf("health: %d", res.Status)
	}
}

func TestAuthLifecycle(t *testing.T) {
	r := newRouter()
	email := uniqueEmail()

	for name, body := range map[string]map[string]string{
		"bad email":     {"email": "nope", "password": "Passw0rdTest"},
		"short pass":    {"email": email, "password": "a1"},
		"no digit":      {"email": email, "password": "onlyletters"},
		"missing field": {"email": email},
	} {
		if res := do(t, r, http.MethodPost, "/auth/register", body, ""); res.Status != http.StatusBadRequest {
			t.Errorf("%s: expected 400 got %d", name, res.Status)
		}
	}

	u := register(t, r)
	if res := do(t, r, http.MethodPost, "/auth/register", map[string]string{"email": "  " + u.Email + " ", "password": "Passw0rdTest"}, ""); res.Status != http.StatusConflict {
		t.Fatalf("duplicate (normalised) email: expected 409 got %d", res.Status)
	}

	if res := do(t, r, http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "wrong-pass1"}, ""); res.Status != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/auth/login", map[string]string{"email": "nobody@test.example", "password": "Passw0rdTest1"}, ""); res.Status != http.StatusUnauthorized {
		t.Fatalf("unknown user: %d", res.Status)
	}

	login := do(t, r, http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "Passw0rdTest"}, "")
	if login.Status != 200 {
		t.Fatalf("login: %d %s", login.Status, login.Body)
	}
	var tok struct{ AccessToken, RefreshToken string }
	login.json(t, &tok)

	if res := do(t, r, http.MethodGet, "/me", nil, ""); res.Status != 401 {
		t.Fatalf("/me without token: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/me", nil, "garbage"); res.Status != 401 {
		t.Fatalf("/me with garbage: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/me", nil, tok.AccessToken); res.Status != 200 {
		t.Fatalf("/me: %d", res.Status)
	}

	// Refresh rotates the refresh token: the old one stops working.
	ref := do(t, r, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": tok.RefreshToken}, "")
	if ref.Status != 200 {
		t.Fatalf("refresh: %d %s", ref.Status, ref.Body)
	}
	var next struct{ AccessToken, RefreshToken string }
	ref.json(t, &next)
	if res := do(t, r, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": tok.RefreshToken}, ""); res.Status != 401 {
		t.Fatalf("replayed refresh token should fail: %d", res.Status)
	}

	// Logout revokes the session immediately, even for a still-valid access token.
	if res := do(t, r, http.MethodPost, "/auth/logout", map[string]string{"refreshToken": next.RefreshToken}, ""); res.Status != 204 {
		t.Fatalf("logout: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/me", nil, next.AccessToken); res.Status != 401 {
		t.Fatalf("revoked session still accepted: %d", res.Status)
	}
}

func TestPasswordRecovery(t *testing.T) {
	r := newRouter()
	u := register(t, r)

	if res := do(t, r, http.MethodPost, "/auth/password/forgot", map[string]string{"email": "ghost@test.example"}, ""); res.Status != http.StatusAccepted {
		t.Fatalf("unknown email must look identical: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/auth/password/forgot", map[string]string{"email": u.Email}, ""); res.Status != http.StatusAccepted {
		t.Fatalf("forgot: %d", res.Status)
	}
	code := testMail.codeFor(u.Email)
	if code == "" {
		t.Fatal("no code emailed")
	}
	if res := do(t, r, http.MethodPost, "/auth/password/reset", map[string]string{"email": u.Email, "code": "00000000", "newPassword": "NewPassw0rd"}, ""); res.Status != 400 {
		t.Fatalf("wrong code: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/auth/password/reset", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassw0rd"}, ""); res.Status != 204 {
		t.Fatalf("reset: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/auth/password/reset", map[string]string{"email": u.Email, "code": code, "newPassword": "Another1Pass"}, ""); res.Status != 400 {
		t.Fatalf("code reuse must fail: %d", res.Status)
	}
	if res := do(t, r, http.MethodGet, "/me", nil, u.Token); res.Status != 401 {
		t.Fatalf("old sessions must be revoked after reset: %d", res.Status)
	}
	if res := do(t, r, http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "NewPassw0rd"}, ""); res.Status != 200 {
		t.Fatalf("login with new password: %d", res.Status)
	}
}

func TestResetCodeAttemptsAreLimited(t *testing.T) {
	r := newRouter()
	u := register(t, r)
	do(t, r, http.MethodPost, "/auth/password/forgot", map[string]string{"email": u.Email}, "")
	code := testMail.codeFor(u.Email)
	for i := 0; i < 5; i++ {
		do(t, r, http.MethodPost, "/auth/password/reset", map[string]string{"email": u.Email, "code": "11111111", "newPassword": "NewPassw0rd"}, "")
	}
	if res := do(t, r, http.MethodPost, "/auth/password/reset", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassw0rd"}, ""); res.Status != 400 {
		t.Fatalf("brute force protection: correct code must be refused after 5 failures, got %d", res.Status)
	}
}
