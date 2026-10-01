package app_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.want(e.do("GET", "/health", "", nil), 200)
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)
	email := "  Alice@Test.Invalid "
	r := e.want(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123"}), 201).json()
	if r["user"].(map[string]any)["email"] != "alice@test.invalid" {
		t.Fatalf("email must be trimmed and lower-cased: %v", r["user"])
	}
	if _, leaked := r["user"].(map[string]any)["passwordHash"]; leaked {
		t.Fatal("password hash leaked")
	}
	var hash string
	_ = e.pool.QueryRow(t.Context(), `SELECT password_hash FROM users WHERE email='alice@test.invalid'`).Scan(&hash)
	if len(hash) < 50 || hash == "Password123" || hash[:4] != "$2a$" {
		t.Fatalf("password must be stored as bcrypt, got %q", hash)
	}

	e.want(e.do("POST", "/auth/register", "", map[string]string{"email": "alice@test.invalid", "password": "Password123"}), 409)
	e.want(e.do("POST", "/auth/register", "", map[string]string{"email": "bob@test.invalid", "password": "short"}), 400)
	e.want(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Password123"}), 400)
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	e.want(e.do("POST", "/auth/register", "", map[string]string{"email": "bob@test.invalid", "password": string(long)}), 400)

	e.want(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@test.invalid", "password": "wrong-password"}), 401)
	e.want(e.do("POST", "/auth/login", "", map[string]string{"email": "ghost@test.invalid", "password": "Password123"}), 401)
	login := e.want(e.do("POST", "/auth/login", "", map[string]string{"email": "ALICE@test.invalid", "password": "Password123"}), 200).json()
	access, refresh := login["accessToken"].(string), login["refreshToken"].(string)

	me := e.want(e.do("GET", "/me", access, nil), 200).json()
	if me["email"] != "alice@test.invalid" {
		t.Fatalf("me: %v", me)
	}
	e.want(e.do("GET", "/me", "", nil), 401)
	e.want(e.do("GET", "/me", "garbage", nil), 401)

	rot := e.want(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh}), 200).json()
	if rot["refreshToken"] == refresh {
		t.Fatal("refresh token must rotate")
	}
	e.want(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh}), 401)
	e.want(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": rot["refreshToken"].(string)}), 204)
	e.want(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": rot["refreshToken"].(string)}), 401)
}

func TestAccountRecovery(t *testing.T) {
	e := newEnv(t)
	u := e.register("recover")
	// Unknown emails get the same answer: no account enumeration.
	e.want(e.do("POST", "/auth/forgot", "", map[string]string{"email": "nobody@test.invalid"}), 202)
	e.want(e.do("POST", "/auth/forgot", "", map[string]string{"email": u.Email}), 202)
	code := e.mailer.code(u.Email)
	if len(code) != 8 {
		t.Fatalf("expected an 8 char code in the email, got %q", code)
	}
	e.want(e.do("POST", "/auth/reset", "", map[string]string{"email": u.Email, "code": "AAAAAAAA", "newPassword": "NewPassword456"}), 400)
	e.want(e.do("POST", "/auth/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "short"}), 400)
	e.want(e.do("POST", "/auth/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}), 204)
	// The code is single-use and old sessions are revoked.
	e.want(e.do("POST", "/auth/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Another789pass"}), 400)
	e.want(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), 401)
	e.want(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": u.Password}), 401)
	e.want(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword456"}), 200)
}

func TestRecoveryCodeBruteForceLimited(t *testing.T) {
	e := newEnv(t)
	u := e.register("brute")
	e.want(e.do("POST", "/auth/forgot", "", map[string]string{"email": u.Email}), 202)
	code := e.mailer.code(u.Email)
	for i := 0; i < 5; i++ {
		e.want(e.do("POST", "/auth/reset", "", map[string]string{"email": u.Email, "code": "WRONGCOD", "newPassword": "NewPassword456"}), 400)
	}
	// After 5 failed attempts even the right code is refused.
	e.want(e.do("POST", "/auth/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}), 400)
}

func TestJWTValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("jwt")
	sign := func(method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	base := func() jwt.MapClaims {
		return jwt.MapClaims{"uid": u.ID, "sid": "00000000-0000-0000-0000-000000000000", "exp": time.Now().Add(time.Minute).Unix()}
	}
	e.want(e.do("GET", "/me", sign(jwt.SigningMethodHS256, jwtSecret, base()), nil), 200)
	e.want(e.do("GET", "/me", sign(jwt.SigningMethodHS512, jwtSecret, base()), nil), 401)
	e.want(e.do("GET", "/me", sign(jwt.SigningMethodHS256, []byte("another-secret-another-secret-123456"), base()), nil), 401)
	expired := base()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	e.want(e.do("GET", "/me", sign(jwt.SigningMethodHS256, jwtSecret, expired), nil), 401)
	noExp := base()
	delete(noExp, "exp")
	e.want(e.do("GET", "/me", sign(jwt.SigningMethodHS256, jwtSecret, noExp), nil), 401)
	none := sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, base())
	e.want(e.do("GET", "/me", none, nil), 401)
	noUID := base()
	delete(noUID, "uid")
	e.want(e.do("GET", "/me", sign(jwt.SigningMethodHS256, jwtSecret, noUID), nil), 401)
}
