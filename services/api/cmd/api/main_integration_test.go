package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	e := newTestEnv(t)
	resp := e.expect(t, e.do(t, http.MethodGet, "/health", nil, ""), http.StatusOK, "health")
	if !strings.Contains(string(resp.Body), `"status":"ok"`) {
		t.Fatalf("unexpected health body: %s", string(resp.Body))
	}
}

func TestHealthPropagatesRequestID(t *testing.T) {
	e := newTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "req-integration-fixed-id")
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "req-integration-fixed-id" {
		t.Fatalf("expected request id echo, got %q", got)
	}
}

func TestHealthReturns503WhenDBUnavailable(t *testing.T) {
	e := newTestEnv(t)
	e.pool.Close()
	resp := e.do(t, http.MethodGet, "/health", nil, "")
	if resp.Status != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Status)
	}
}

func TestUnauthorizedEndpoints(t *testing.T) {
	e := newTestEnv(t)
	const id = "11111111-1111-1111-1111-111111111111"
	for _, c := range []struct{ method, path string }{
		{"GET", "/me"}, {"GET", "/me/profile"}, {"GET", "/me/preferences"}, {"GET", "/me/photos"},
		{"GET", "/discover"}, {"POST", "/swipes"}, {"GET", "/matches"}, {"DELETE", "/matches/" + id},
		{"GET", "/conversations"}, {"GET", "/conversations/" + id + "/messages"}, {"POST", "/conversations/" + id + "/messages"},
		{"GET", "/notifications"}, {"GET", "/blocks"}, {"POST", "/reports"}, {"GET", "/profiles/" + id},
		{"GET", "/photos/" + id + "/file"}, {"DELETE", "/me"}, {"GET", "/admin/reports"}, {"POST", "/ws/ticket"},
		// legacy boilerplate surfaces must stay gone
		{"GET", "/users"},
	} {
		resp := e.do(t, c.method, c.path, nil, "")
		want := http.StatusUnauthorized
		if c.path == "/users" {
			want = http.StatusNotFound
		}
		if resp.Status != want {
			t.Fatalf("%s %s expected %d, got %d", c.method, c.path, want, resp.Status)
		}
	}
}

func TestAuthLifecycle(t *testing.T) {
	e := newTestEnv(t)
	u := e.register(t, "auth")

	me := e.expect(t, e.do(t, http.MethodGet, "/me", nil, u.Token), http.StatusOK, "me")
	var meBody struct {
		Email           string `json:"email"`
		ProfileComplete bool   `json:"profileComplete"`
		Role            string `json:"role"`
	}
	me.decode(t, &meBody)
	if meBody.Email != u.Email || meBody.ProfileComplete || meBody.Role != "user" {
		t.Fatalf("unexpected /me: %+v", meBody)
	}

	// duplicate email (case-insensitive) and weak/oversized passwords
	e.expect(t, e.do(t, http.MethodPost, "/auth/register", map[string]string{"email": strings.ToUpper(u.Email), "password": "Password123"}, ""), http.StatusConflict, "duplicate register")
	e.expect(t, e.do(t, http.MethodPost, "/auth/register", map[string]string{"email": "x@test.invalid", "password": "short"}, ""), http.StatusBadRequest, "short password")
	e.expect(t, e.do(t, http.MethodPost, "/auth/register", map[string]string{"email": "y@test.invalid", "password": strings.Repeat("a", 73)}, ""), http.StatusBadRequest, "73-char password")
	e.expect(t, e.do(t, http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "WrongPassword1"}, ""), http.StatusUnauthorized, "bad login")
	e.expect(t, e.do(t, http.MethodPost, "/auth/login", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": u.Password}, ""), http.StatusOK, "login normalizes email")

	refreshed := e.expect(t, e.do(t, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": u.RefreshToken}, ""), http.StatusOK, "refresh")
	var tokens struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	refreshed.decode(t, &tokens)
	if tokens.RefreshToken == u.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	e.expect(t, e.do(t, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": u.RefreshToken}, ""), http.StatusUnauthorized, "old refresh token reuse")

	e.expect(t, e.do(t, http.MethodPost, "/auth/logout", map[string]string{"refreshToken": tokens.RefreshToken}, ""), http.StatusNoContent, "logout")
	// logout kills the session server-side: the still-unexpired access token stops working
	e.expect(t, e.do(t, http.MethodGet, "/me", nil, tokens.AccessToken), http.StatusUnauthorized, "access token after logout")
	e.expect(t, e.do(t, http.MethodGet, "/me", nil, "garbage"), http.StatusUnauthorized, "garbage token")
}

func TestPasswordChangeAndRecovery(t *testing.T) {
	e := newTestEnv(t)
	u := e.register(t, "pw")

	e.expect(t, e.do(t, http.MethodPost, "/me/password", map[string]string{"currentPassword": "nope-nope", "newPassword": "NewPassword456"}, u.Token), http.StatusForbidden, "wrong current password")
	e.expect(t, e.do(t, http.MethodPost, "/me/password", map[string]string{"currentPassword": u.Password, "newPassword": "NewPassword456"}, u.Token), http.StatusNoContent, "change password")
	e.expect(t, e.do(t, http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": u.Password}, ""), http.StatusUnauthorized, "old password")

	// recovery: unknown address answers identically and sends nothing
	unknown := e.expect(t, e.do(t, http.MethodPost, "/auth/password/forgot", map[string]string{"email": "nobody@test.invalid"}, ""), http.StatusAccepted, "forgot unknown")
	known := e.expect(t, e.do(t, http.MethodPost, "/auth/password/forgot", map[string]string{"email": u.Email}, ""), http.StatusAccepted, "forgot known")
	if string(unknown.Body) != string(known.Body) {
		t.Fatal("forgot-password must not reveal whether an account exists")
	}
	mail := e.mailer.waitFor(t, u.Email)
	code := ""
	for _, line := range strings.Split(mail, "\n") {
		if i := strings.Index(line, "récupération Lumen : "); i >= 0 {
			code = strings.TrimSpace(line[i+len("récupération Lumen : "):])
		}
	}
	if len(code) != 10 {
		t.Fatalf("expected a 10-char code in mail, got %q (%s)", code, mail)
	}

	e.expect(t, e.do(t, http.MethodPost, "/auth/password/reset", map[string]string{"code": "WRONGCODE1", "newPassword": "ResetPassword789"}, ""), http.StatusBadRequest, "wrong code")
	e.expect(t, e.do(t, http.MethodPost, "/auth/password/reset", map[string]string{"code": strings.ToLower(code), "newPassword": "ResetPassword789"}, ""), http.StatusNoContent, "reset")
	e.expect(t, e.do(t, http.MethodPost, "/auth/password/reset", map[string]string{"code": code, "newPassword": "AnotherPassword1"}, ""), http.StatusBadRequest, "code is single use")
	e.expect(t, e.do(t, http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "ResetPassword789"}, ""), http.StatusOK, "login with reset password")
	// reset revoked the session that existed before
	e.expect(t, e.do(t, http.MethodGet, "/me", nil, u.Token), http.StatusUnauthorized, "old session after reset")
}

func TestSuspendedAccountCannotUseTheAPI(t *testing.T) {
	e := newTestEnv(t)
	u := e.register(t, "susp")
	if _, err := e.pool.Exec(t.Context(), `UPDATE users SET status = 'suspended' WHERE id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	e.expect(t, e.do(t, http.MethodGet, "/me", nil, u.Token), http.StatusForbidden, "suspended token")
	e.expect(t, e.do(t, http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": u.Password}, ""), http.StatusForbidden, "suspended login")
	e.expect(t, e.do(t, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": u.RefreshToken}, ""), http.StatusForbidden, "suspended refresh")
}

func TestRateLimitingOnAuthEndpoints(t *testing.T) {
	e := newTestEnv(t)
	e.app.rateLimit = true
	e.rebuild()
	var last testResponse
	for i := 0; i < 12; i++ {
		last = e.do(t, http.MethodPost, "/auth/login", map[string]string{"email": "nobody@test.invalid", "password": "Password123"}, "")
		if last.Status == http.StatusTooManyRequests {
			if last.Header.Get("Retry-After") == "" {
				t.Fatal("429 must carry Retry-After")
			}
			return
		}
	}
	t.Fatalf("expected 429 after repeated logins, last status %d", last.Status)
}

func TestOversizedJSONBodyRejected(t *testing.T) {
	e := newTestEnv(t)
	u := e.register(t, "big")
	big := strings.Repeat("a", 2<<20)
	resp := e.do(t, http.MethodPut, "/me/profile", map[string]string{"firstName": "A", "bio": big}, u.Token)
	if resp.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 for >1MB JSON, got %d", resp.Status)
	}
}
