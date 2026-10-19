package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/server"

	"github.com/golang-jwt/jwt/v5"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	if r := e.do("GET", "/health", "", nil); r.Status != 200 {
		t.Fatalf("health: %d", r.Status)
	}
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	u := e.register("Alice")

	if r := e.do("POST", "/auth/register", "", map[string]string{"email": " " + strings.ToUpper(u.Email) + " ", "password": "Password123"}); r.Status != 409 || r.code() != "email_taken" {
		t.Fatalf("duplicate email (case/space-insensitive) should conflict, got %d %s", r.Status, r.Body)
	}
	if r := e.do("POST", "/auth/register", "", map[string]string{"email": "short@test.invalid", "password": "short"}); r.Status != 400 {
		t.Fatalf("short password: %d", r.Status)
	}
	if r := e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Password123"}); r.Status != 400 {
		t.Fatalf("bad email: %d", r.Status)
	}

	var hash string
	if err := e.pool.QueryRow(t.Context(), `SELECT password_hash FROM users WHERE id=$1`, u.ID).Scan(&hash); err != nil || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("password must be stored as bcrypt, got %q (%v)", hash, err)
	}

	if r := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "wrong-password"}); r.Status != 401 {
		t.Fatalf("wrong password: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": "nobody@test.invalid", "password": "Password123"}); r.Status != 401 {
		t.Fatalf("unknown email: %d", r.Status)
	}
	login := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": u.Password})
	if login.Status != 200 {
		t.Fatalf("login: %d", login.Status)
	}
	var tokens struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	login.decode(t, &tokens)

	me := e.do("GET", "/me", tokens.AccessToken, nil)
	if me.Status != 200 {
		t.Fatalf("me: %d", me.Status)
	}
	if strings.Contains(string(me.Body), "password") {
		t.Fatalf("/me must not leak password data: %s", me.Body)
	}

	rot := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": tokens.RefreshToken})
	if rot.Status != 200 {
		t.Fatalf("refresh: %d", rot.Status)
	}
	var next struct {
		RefreshToken string `json:"refreshToken"`
	}
	rot.decode(t, &next)
	if next.RefreshToken == tokens.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": tokens.RefreshToken}); r.Status != 401 {
		t.Fatalf("old refresh token must be rejected, got %d", r.Status)
	}
	if r := e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": next.RefreshToken}); r.Status != 204 {
		t.Fatalf("logout: %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": next.RefreshToken}); r.Status != 401 {
		t.Fatalf("refresh after logout must fail, got %d", r.Status)
	}
}

func TestTokenValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("Bob")

	if r := e.do("GET", "/me", "", nil); r.Status != 401 {
		t.Fatalf("missing token: %d", r.Status)
	}
	if r := e.do("GET", "/me", "garbage", nil); r.Status != 401 {
		t.Fatalf("garbage token: %d", r.Status)
	}

	sign := func(method jwt.SigningMethod, secret any, claims jwt.MapClaims) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(secret)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	valid := jwt.MapClaims{"uid": u.ID, "sid": "s", "exp": time.Now().Add(time.Minute).Unix()}
	if r := e.do("GET", "/me", sign(jwt.SigningMethodHS256, []byte(testSecret), valid), nil); r.Status != 200 {
		t.Fatalf("sanity: valid token should pass, got %d", r.Status)
	}
	cases := map[string]string{
		"wrong secret": sign(jwt.SigningMethodHS256, []byte("another-secret-another-secret-0000"), valid),
		"wrong alg":    sign(jwt.SigningMethodHS512, []byte(testSecret), valid),
		"none alg":     sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid),
		"expired":      sign(jwt.SigningMethodHS256, []byte(testSecret), jwt.MapClaims{"uid": u.ID, "sid": "s", "exp": time.Now().Add(-time.Minute).Unix()}),
		"no expiry":    sign(jwt.SigningMethodHS256, []byte(testSecret), jwt.MapClaims{"uid": u.ID, "sid": "s"}),
		"no uid":       sign(jwt.SigningMethodHS256, []byte(testSecret), jwt.MapClaims{"sid": "s", "exp": time.Now().Add(time.Minute).Unix()}),
	}
	for name, tok := range cases {
		if r := e.do("GET", "/me", tok, nil); r.Status != 401 {
			t.Errorf("%s: expected 401, got %d", name, r.Status)
		}
	}
}

func TestPasswordRecovery(t *testing.T) {
	e := newEnv(t)
	u := e.register("Carol")

	if r := e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "ghost@test.invalid"}); r.Status != http.StatusAccepted {
		t.Fatalf("unknown email must look identical, got %d", r.Status)
	}
	if e.mailer.code("ghost@test.invalid") != "" {
		t.Fatal("no code should be issued for unknown emails")
	}
	if r := e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}); r.Status != http.StatusAccepted {
		t.Fatalf("forgot: %d", r.Status)
	}
	code := e.mailer.code(u.Email)
	if len(code) != 10 {
		t.Fatalf("expected a 10 char code, got %q", code)
	}

	reset := func(code, pw string) resp {
		return e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": pw})
	}
	if r := reset("WRONGCODE1", "NewPassword456"); r.Status != 400 || r.code() != "invalid_code" {
		t.Fatalf("wrong code: %d %s", r.Status, r.Body)
	}
	if r := reset(strings.ToLower(code), "NewPassword456"); r.Status != 204 {
		t.Fatalf("correct code (case-insensitive) should work: %d %s", r.Status, r.Body)
	}
	if r := reset(code, "Another789xx"); r.Status != 400 {
		t.Fatalf("a code is single use, got %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}); r.Status != 401 {
		t.Fatalf("existing sessions must be revoked, got %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": u.Password}); r.Status != 401 {
		t.Fatalf("old password must stop working, got %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword456"}); r.Status != 200 {
		t.Fatalf("new password should work, got %d", r.Status)
	}
}

func TestPasswordRecoveryBruteForceLockout(t *testing.T) {
	e := newEnv(t)
	u := e.register("Dave")
	e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email})
	code := e.mailer.code(u.Email)

	// Attempts are rate limited per IP, so count failures directly in the table
	// after a few wrong guesses and verify the right code is then refused.
	for i := 0; i < 5; i++ {
		e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": "AAAAAAAAA" + string(rune('A'+i)), "newPassword": "NewPassword456"})
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}); r.Status != 400 {
		t.Fatalf("code must be burned after too many failures, got %d", r.Status)
	}
}

func TestAuthRateLimit(t *testing.T) {
	e := newEnvWithLimits(t, server.Limits{})
	limited := false
	for i := 0; i < 20; i++ {
		r := e.do("POST", "/auth/login", "", map[string]string{"email": "x@test.invalid", "password": "Password123"})
		if r.Status == http.StatusTooManyRequests {
			limited = true
			if r.Header.Get("Retry-After") == "" {
				t.Fatal("429 must carry Retry-After")
			}
			break
		}
	}
	if !limited {
		t.Fatal("login endpoint should rate limit bursts")
	}
}
