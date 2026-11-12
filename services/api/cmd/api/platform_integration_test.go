package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	env := newEnv(t)
	r := env.do(t, http.MethodGet, "/health", nil, "")
	expect(t, r, http.StatusOK, "health")
	if !strings.Contains(string(r.Body), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", r)
	}
}

func TestHealthPropagatesRequestID(t *testing.T) {
	env := newEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "req-integration-fixed-id")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "req-integration-fixed-id" {
		t.Fatalf("expected request id echo, got %q", got)
	}
}

func TestHealthReturns503WhenDBUnavailable(t *testing.T) {
	env := newEnv(t)
	env.pool.Close()
	r := env.do(t, http.MethodGet, "/health", nil, "")
	expect(t, r, http.StatusServiceUnavailable, "health without db")
}

func TestUnauthorizedEndpoints(t *testing.T) {
	env := newEnv(t)
	for _, path := range []string{"/me", "/users", "/profile", "/discover", "/matches", "/chats", "/notifications", "/blocks", "/posts"} {
		expect(t, env.do(t, http.MethodGet, path, nil, ""), http.StatusUnauthorized, "GET "+path+" without token")
		expect(t, env.do(t, http.MethodGet, path, nil, "not-a-jwt"), http.StatusUnauthorized, "GET "+path+" with bad token")
	}
}

func TestAuthFlowAndSessionRevocation(t *testing.T) {
	env := newEnv(t)
	email := uniqueEmail("auth")
	env.emails = append(env.emails, email)

	expect(t, env.do(t, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": "short"}, ""), http.StatusBadRequest, "weak password")
	expect(t, env.do(t, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": "onlyletters"}, ""), http.StatusBadRequest, "no digit")

	reg := env.do(t, http.MethodPost, "/auth/register", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": testPassword}, "")
	expect(t, reg, http.StatusCreated, "register")
	a := decode[authPayload](t, reg)
	if a.User.Email != email {
		t.Fatalf("email must be normalized, got %q", a.User.Email)
	}
	expect(t, env.do(t, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": testPassword}, ""), http.StatusConflict, "duplicate register")
	expect(t, env.do(t, http.MethodPost, "/auth/login", map[string]string{"email": email, "password": "Wrong12345"}, ""), http.StatusUnauthorized, "bad login")

	login := env.do(t, http.MethodPost, "/auth/login", map[string]string{"email": email, "password": testPassword}, "")
	expect(t, login, http.StatusOK, "login")
	session := decode[authPayload](t, login)
	expect(t, env.do(t, http.MethodGet, "/me", nil, session.AccessToken), http.StatusOK, "me")

	refreshed := env.do(t, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": session.RefreshToken}, "")
	expect(t, refreshed, http.StatusOK, "refresh")
	next := decode[authPayload](t, refreshed)
	expect(t, env.do(t, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": session.RefreshToken}, ""), http.StatusUnauthorized, "old refresh token reuse")

	// Logout must revoke the access token immediately, not after its TTL.
	expect(t, env.do(t, http.MethodPost, "/auth/logout", map[string]string{"refreshToken": next.RefreshToken}, ""), http.StatusNoContent, "logout")
	expect(t, env.do(t, http.MethodGet, "/me", nil, next.AccessToken), http.StatusUnauthorized, "access token after logout")
	expect(t, env.do(t, http.MethodGet, "/me", nil, a.AccessToken), http.StatusOK, "other session still valid")
}

func TestPasswordResetAndChange(t *testing.T) {
	env := newEnv(t)
	m := env.register(t, "reset")

	expect(t, env.do(t, http.MethodPost, "/auth/password/forgot", map[string]string{"email": "nobody_" + m.Email}, ""), http.StatusAccepted, "forgot unknown email")
	expect(t, env.do(t, http.MethodPost, "/auth/password/forgot", map[string]string{"email": m.Email}, ""), http.StatusAccepted, "forgot")
	code := env.mail.codeFor(t, m.Email)

	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	expect(t, env.do(t, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": wrong, "newPassword": "NewPassword456"}, ""), http.StatusBadRequest, "wrong code")
	expect(t, env.do(t, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "newPassword": "NewPassword456"}, ""), http.StatusNoContent, "reset")
	expect(t, env.do(t, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "newPassword": "NewPassword789"}, ""), http.StatusBadRequest, "code reuse")

	expect(t, env.do(t, http.MethodGet, "/me", nil, m.Token), http.StatusUnauthorized, "sessions revoked after reset")
	expect(t, env.do(t, http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": testPassword}, ""), http.StatusUnauthorized, "old password")
	login := env.do(t, http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": "NewPassword456"}, "")
	expect(t, login, http.StatusOK, "login with new password")
	token := decode[authPayload](t, login).AccessToken

	expect(t, env.do(t, http.MethodPut, "/me/password", map[string]string{"currentPassword": "bad", "newPassword": "Another123"}, token), http.StatusForbidden, "change with wrong current")
	expect(t, env.do(t, http.MethodPut, "/me/password", map[string]string{"currentPassword": "NewPassword456", "newPassword": "Another123"}, token), http.StatusNoContent, "change password")
	expect(t, env.do(t, http.MethodGet, "/me", nil, token), http.StatusOK, "current session kept after change")
}

func TestPasswordResetLocksAfterTooManyAttempts(t *testing.T) {
	env := newEnv(t)
	m := env.register(t, "resetlock")
	expect(t, env.do(t, http.MethodPost, "/auth/password/forgot", map[string]string{"email": m.Email}, ""), http.StatusAccepted, "forgot")
	code := env.mail.codeFor(t, m.Email)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 5; i++ {
		env.do(t, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": wrong, "newPassword": "NewPassword456"}, "")
	}
	expect(t, env.do(t, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "newPassword": "NewPassword456"}, ""), http.StatusBadRequest, "correct code after lockout")
}

func TestAuthRateLimit(t *testing.T) {
	env := newEnv(t)
	limits := generousLimits()
	limits.Auth = newTightLimiter(3)
	a := &app{dbPool: env.pool, jwtSecret: []byte(testJWTSecret), mailer: env.mail, hub: env.hub, limits: limits}
	a.storage = mustStorage(t)
	router := setupRouter(a)
	var last int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"x@example.test","password":"Password123"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		last = rec.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding auth quota, got %d", last)
	}
}

func TestPostsCommentsAndFilesStillWork(t *testing.T) {
	env := newEnv(t)
	m := env.register(t, "content")

	post := env.do(t, http.MethodPost, "/posts", map[string]string{"title": "Hello", "body": "Generic body"}, m.Token)
	expect(t, post, http.StatusCreated, "create post")
	postID := decode[struct {
		ID string `json:"id"`
	}](t, post).ID
	expect(t, env.do(t, http.MethodGet, "/posts", nil, m.Token), http.StatusOK, "list posts")
	expect(t, env.do(t, http.MethodPost, "/posts/"+postID+"/comments", map[string]string{"content": "Nice"}, m.Token), http.StatusCreated, "comment")
	expect(t, env.do(t, http.MethodGet, "/posts/"+postID+"/comments", nil, m.Token), http.StatusOK, "list comments")
	expect(t, env.do(t, http.MethodPost, "/files", map[string]any{"filename": "a.txt", "mimeType": "text/plain", "sizeBytes": 3, "storageKey": "k/" + m.ID}, m.Token), http.StatusCreated, "file")
	expect(t, env.do(t, http.MethodGet, "/files", nil, m.Token), http.StatusOK, "list files")
}
