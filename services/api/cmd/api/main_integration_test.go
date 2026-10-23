package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	e := newEnv(t)
	r := e.expect(e.do(http.MethodGet, "/health", nil, ""), http.StatusOK)
	if !strings.Contains(string(r.Body), `"status":"ok"`) {
		t.Fatalf("unexpected health body: %s", r.Body)
	}
}

func TestHealthPropagatesRequestID(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "req-integration-fixed-id")
	r := e.serve(req)
	if r.Header.Get("X-Request-ID") != "req-integration-fixed-id" {
		t.Fatalf("request id not propagated: %q", r.Header.Get("X-Request-ID"))
	}
}

func TestHealthReturns503WhenDBUnavailable(t *testing.T) {
	e := newEnv(t)
	e.pool.Close()
	r := e.do(http.MethodGet, "/health", nil, "")
	if r.Status != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", r.Status)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	e := newEnv(t)
	r := e.do(http.MethodGet, "/health", nil, "")
	if r.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff header")
	}
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	e := newEnv(t)
	routes := []struct{ method, path string }{
		{"GET", "/me"}, {"DELETE", "/me"}, {"GET", "/me/profile"}, {"PUT", "/me/profile"},
		{"GET", "/me/preferences"}, {"GET", "/me/photos"}, {"POST", "/me/photos"},
		{"GET", "/photos/123e4567-e89b-12d3-a456-426614174000"}, {"GET", "/discover"},
		{"GET", "/profiles/123e4567-e89b-12d3-a456-426614174000"}, {"POST", "/swipes"},
		{"GET", "/matches"}, {"GET", "/matches/123e4567-e89b-12d3-a456-426614174000/messages"},
		{"POST", "/matches/123e4567-e89b-12d3-a456-426614174000/messages"},
		{"POST", "/blocks"}, {"POST", "/reports"}, {"GET", "/notifications"}, {"GET", "/interests"},
	}
	for _, rt := range routes {
		if r := e.do(rt.method, rt.path, nil, ""); r.Status != http.StatusUnauthorized {
			t.Errorf("%s %s without token: expected 401, got %d", rt.method, rt.path, r.Status)
		}
		if r := e.do(rt.method, rt.path, nil, "garbage.token.value"); r.Status != http.StatusUnauthorized {
			t.Errorf("%s %s with bad token: expected 401, got %d", rt.method, rt.path, r.Status)
		}
	}
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register("auth")

	me := e.expect(e.do(http.MethodGet, "/me", nil, u.Token), http.StatusOK).JSON(t)
	if me["email"] != u.Email {
		t.Fatalf("unexpected /me: %v", me)
	}
	onboarding := me["onboarding"].(map[string]any)
	if onboarding["completed"] != false || onboarding["hasProfile"] != false {
		t.Fatalf("new accounts must not be onboarded: %v", onboarding)
	}
	if _, leaked := me["passwordHash"]; leaked {
		t.Fatal("password hash leaked")
	}

	e.expect(e.do(http.MethodPost, "/auth/register", map[string]string{"email": strings.ToUpper(u.Email), "password": testPassword}, ""), http.StatusConflict)
	e.expect(e.do(http.MethodPost, "/auth/register", map[string]string{"email": uniqueEmail("weak"), "password": "short"}, ""), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/register", map[string]string{"email": "not-an-email", "password": testPassword}, ""), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": "WrongPassword1"}, ""), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", map[string]string{"email": "nobody@itest.example.com", "password": testPassword}, ""), http.StatusUnauthorized)

	login := e.expect(e.do(http.MethodPost, "/auth/login", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": testPassword}, ""), http.StatusOK).JSON(t)
	if login["accessToken"] == "" || login["refreshToken"] == "" {
		t.Fatal("missing tokens")
	}

	// Refresh rotates: the new token works once, the old one is dead.
	refreshed := e.expect(e.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": u.Refresh}, ""), http.StatusOK).JSON(t)
	newRefresh := refreshed["refreshToken"].(string)
	if newRefresh == u.Refresh {
		t.Fatal("refresh token must rotate")
	}
	e.expect(e.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": u.Refresh}, ""), http.StatusUnauthorized)
	newAccess := refreshed["accessToken"].(string)
	e.expect(e.do(http.MethodGet, "/me", nil, newAccess), http.StatusOK)

	// Logout kills the session immediately, including its access token.
	e.expect(e.do(http.MethodPost, "/auth/logout", map[string]string{"refreshToken": newRefresh}, ""), http.StatusNoContent)
	e.expect(e.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": newRefresh}, ""), http.StatusUnauthorized)
	e.expect(e.do(http.MethodGet, "/me", nil, newAccess), http.StatusUnauthorized)
}

func TestConcurrentRefreshOnlyOneWins(t *testing.T) {
	e := newEnv(t)
	u := e.register("race")
	results := make(chan int, 8)
	for i := 0; i < 8; i++ {
		go func() {
			results <- e.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": u.Refresh}, "").Status
		}()
	}
	ok := 0
	for i := 0; i < 8; i++ {
		if <-results == http.StatusOK {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("exactly one concurrent refresh must succeed, got %d", ok)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register("reset")

	// Unknown emails get the same answer and no mail.
	e.expect(e.do(http.MethodPost, "/auth/forgot-password", map[string]string{"email": "ghost@itest.example.com"}, ""), http.StatusAccepted)
	e.expect(e.do(http.MethodPost, "/auth/forgot-password", map[string]string{"email": u.Email}, ""), http.StatusAccepted)
	code := e.mail.codeFor(t, u.Email)

	newPassword := "BrandNewPass99"
	e.expect(e.do(http.MethodPost, "/auth/reset-password", map[string]string{"email": u.Email, "code": "WRONGCODE1", "password": newPassword}, ""), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/reset-password", map[string]string{"email": u.Email, "code": code, "password": "weak"}, ""), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/reset-password", map[string]string{"email": u.Email, "code": strings.ToLower(code), "password": newPassword}, ""), http.StatusNoContent)

	// Code is single-use, old sessions are revoked, old password is dead.
	e.expect(e.do(http.MethodPost, "/auth/reset-password", map[string]string{"email": u.Email, "code": code, "password": newPassword}, ""), http.StatusBadRequest)
	e.expect(e.do(http.MethodGet, "/me", nil, u.Token), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": testPassword}, ""), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", map[string]string{"email": u.Email, "password": newPassword}, ""), http.StatusOK)
}

func TestPasswordResetLocksAfterTooManyWrongCodes(t *testing.T) {
	e := newEnv(t)
	u := e.register("lock")
	e.expect(e.do(http.MethodPost, "/auth/forgot-password", map[string]string{"email": u.Email}, ""), http.StatusAccepted)
	code := e.mail.codeFor(t, u.Email)
	for i := 0; i < 5; i++ {
		e.expect(e.do(http.MethodPost, "/auth/reset-password", map[string]string{"email": u.Email, "code": "AAAAAAAAAA", "password": "BrandNewPass99"}, ""), http.StatusBadRequest)
	}
	// Even the right code is refused now.
	e.expect(e.do(http.MethodPost, "/auth/reset-password", map[string]string{"email": u.Email, "code": code, "password": "BrandNewPass99"}, ""), http.StatusBadRequest)
}

func TestAuthEndpointsAreRateLimited(t *testing.T) {
	e := newEnvWith(t, 3, 100000)
	statuses := []int{}
	for i := 0; i < 5; i++ {
		statuses = append(statuses, e.do(http.MethodPost, "/auth/login", map[string]string{"email": "x@itest.example.com", "password": "Whatever123"}, "").Status)
	}
	if statuses[0] != http.StatusUnauthorized || statuses[4] != http.StatusTooManyRequests {
		t.Fatalf("expected 401s then 429, got %v", statuses)
	}
	r := e.do(http.MethodPost, "/auth/login", map[string]string{"email": "x@itest.example.com", "password": "Whatever123"}, "")
	if r.Header.Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
}

func TestMalformedAndOversizedBodiesAreRejected(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	if r := e.serve(req); r.Status != http.StatusBadRequest {
		t.Fatalf("malformed JSON: expected 400, got %d", r.Status)
	}
	big := `{"email":"` + strings.Repeat("a", 2<<20) + `"}`
	req = httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	if r := e.serve(req); r.Status != http.StatusBadRequest {
		t.Fatalf("oversized JSON: expected 400, got %d", r.Status)
	}
}

func TestAccountDeletionRemovesEverything(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/messages", map[string]string{"content": "salut"}, a.Token), http.StatusCreated)
	e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "spam"}, a.Token), http.StatusCreated)

	e.expect(e.do(http.MethodDelete, "/me", map[string]string{"password": "WrongPassword1"}, a.Token), http.StatusForbidden)
	e.expect(e.do(http.MethodDelete, "/me", map[string]string{"password": testPassword}, a.Token), http.StatusNoContent)

	e.expect(e.do(http.MethodGet, "/me", nil, a.Token), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", map[string]string{"email": a.Email, "password": testPassword}, ""), http.StatusUnauthorized)

	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "preferences": "user_id", "sessions": "user_id",
		"swipes": "swiper_id", "messages": "sender_id", "reports": "reporter_id", "notifications": "user_id"} {
		var n int
		if err := e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s still has %d rows for the deleted user (%v)", table, n, err)
		}
	}
	var matches int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE id = $1`, matchID).Scan(&matches)
	if matches != 0 {
		t.Error("match must be deleted with the account")
	}
	list := e.expect(e.do(http.MethodGet, "/matches", nil, b.Token), http.StatusOK).JSON(t)
	if len(list["matches"].([]any)) != 0 {
		t.Error("the other user must no longer see the deleted match")
	}
}
