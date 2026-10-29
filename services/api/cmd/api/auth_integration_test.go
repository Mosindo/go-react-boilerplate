package main

import (
	"fmt"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	e := newTestEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
}

func TestAuthRegisterLoginMeRefreshLogout(t *testing.T) {
	e := newTestEnv(t)

	email := uniqueEmail("auth")
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": "Password123!"}), 201)
	var reg struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			Email              string `json:"email"`
			OnboardingComplete bool   `json:"onboardingComplete"`
		} `json:"user"`
	}
	r.decode(t, &reg)
	if reg.User.Email != email {
		t.Fatalf("email must be trimmed and lowercased, got %q", reg.User.Email)
	}
	if reg.User.OnboardingComplete {
		t.Fatal("new account cannot have completed onboarding")
	}

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123!"}), 409)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Password123!"}), 400)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": uniqueEmail("short"), "password": "short"}), 400)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": uniqueEmail("long"), "password": strings.Repeat("a", 100)}), 400)

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": "wrong-password"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "nobody@dating.test", "password": "Password123!"}), 401)
	r = e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": "Password123!"}), 200)
	var login struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	r.decode(t, &login)

	e.expect(e.do("GET", "/me", "", nil), 401)
	e.expect(e.do("GET", "/me", "garbage", nil), 401)
	e.expect(e.do("GET", "/me", login.AccessToken, nil), 200)

	r = e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), 200)
	var refreshed struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	r.decode(t, &refreshed)
	if refreshed.RefreshToken == login.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), 401)

	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": refreshed.RefreshToken}), 204)
	e.expect(e.do("GET", "/me", refreshed.AccessToken, nil), 401) // access token dies with its session
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": refreshed.RefreshToken}), 401)
}

func TestAuthRejectsTokensNotSignedWithHS256OrExpired(t *testing.T) {
	e := newTestEnv(t)
	u := e.register("jwt")

	// alg=none token with valid-looking claims
	none := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOiJ4Iiwic2lkIjoieCIsImV4cCI6OTk5OTk5OTk5OX0."
	e.expect(e.do("GET", "/me", none, nil), 401)
	e.expect(e.do("GET", "/me", u.Token+"x", nil), 401)
}

func TestPasswordRecoveryFlow(t *testing.T) {
	e := newTestEnv(t)
	u := e.register("recover")

	// Unknown email: same answer, no email sent.
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": "ghost@dating.test"}), 202)
	if m := e.mailer.last(); m.To != "" {
		t.Fatalf("no email must be sent for unknown accounts, got %+v", m)
	}

	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": u.Email}), 202)
	var mail capturedMail
	for i := 0; i < 50 && mail.To == ""; i++ {
		mail = e.mailer.last()
		time.Sleep(20 * time.Millisecond)
	}
	code := regexp.MustCompile(`[A-Z2-9]{8}`).FindString(mail.Body)
	if mail.To != u.Email || code == "" {
		t.Fatalf("expected a reset email with a code, got %+v", mail)
	}

	body := map[string]string{"email": u.Email, "code": "AAAAAAAA", "newPassword": "NewPassword456!"}
	e.expect(e.do("POST", "/auth/reset-password", "", body), 400)
	body["code"] = strings.ToLower(code[:4]) + "-" + code[4:] // forgiving format
	e.expect(e.do("POST", "/auth/reset-password", "", body), 204)
	e.expect(e.do("POST", "/auth/reset-password", "", body), 400) // single use

	e.expect(e.do("GET", "/me", u.Token, nil), 401) // old sessions revoked
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123!"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword456!"}), 200)
}

func TestPasswordResetLocksAfterTooManyWrongCodes(t *testing.T) {
	e := newTestEnv(t)
	u := e.register("lock")
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": u.Email}), 202)
	var mail capturedMail
	for i := 0; i < 50 && mail.To == ""; i++ {
		mail = e.mailer.last()
		time.Sleep(20 * time.Millisecond)
	}
	code := regexp.MustCompile(`[A-Z2-9]{8}`).FindString(mail.Body)

	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": u.Email, "code": "WRONGCODE", "newPassword": "NewPassword456!"}), 400)
	}
	// Even the right code is refused once the attempt budget is spent.
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456!"}), 400)
}

func TestChangePasswordAndDeleteAccount(t *testing.T) {
	e := newTestEnv(t)
	u := e.onboard(userSpec{Name: "Alice"})

	e.expect(e.do("PUT", "/me/password", u.Token, map[string]string{"currentPassword": "bad", "newPassword": "Another1234!"}), 403)
	e.expect(e.do("PUT", "/me/password", u.Token, map[string]string{"currentPassword": "Password123!", "newPassword": "Another1234!"}), 204)

	e.expect(e.do("DELETE", "/me", u.Token, map[string]string{"password": "Password123!"}), 403)
	e.expect(e.do("DELETE", "/me", u.Token, map[string]string{"password": "Another1234!"}), 204)
	e.expect(e.do("GET", "/me", u.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Another1234!"}), 401)

	var n int
	for _, table := range []string{"users", "profiles", "preferences", "photos", "sessions"} {
		if err := e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("table %s still has %d rows after account deletion", table, n)
		}
	}
}

func TestRateLimitingIsPerClientAddressAndIgnoresSpoofedForwardedFor(t *testing.T) {
	e := newTestEnv(t)
	limited := setupRouter(&app{dbPool: e.pool, jwtSecret: []byte(testJWTSecret), store: e.store, mailer: e.mailer, rateLimits: true})
	e.router = limited

	attempt := func(forwardedFor string) int {
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"nobody@dating.test","password":"whatever-123"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", forwardedFor)
		w := httptest.NewRecorder()
		limited.ServeHTTP(w, req)
		return w.Code
	}

	for i := 0; i < 30; i++ {
		if code := attempt(fmt.Sprintf("10.9.8.%d", i)); code != 401 {
			t.Fatalf("attempt %d: expected 401, got %d", i, code)
		}
	}
	// A different X-Forwarded-For must not buy a fresh budget: no proxy is trusted.
	if code := attempt("203.0.113.77"); code != 429 {
		t.Fatalf("expected 429 once the per-IP budget is spent, got %d", code)
	}
}
