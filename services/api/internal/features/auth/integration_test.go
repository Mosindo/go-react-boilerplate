package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testSecret = []byte("services-test-secret-0123456789abcdef")

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "first.last+tag@example.com"} {
		if !validEmail(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "plain", "a@b", "Name <a@b.co>", "a@@b.co", strings.Repeat("a", 250) + "@b.co"} {
		if validEmail(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
	if normalizeEmail("  Foo@Bar.COM ") != "foo@bar.com" {
		t.Fatal("email must be trimmed and lowercased")
	}
}

func TestAccessTokenIsStrictlyValidated(t *testing.T) {
	user := User{ID: "u1"}
	good, err := signAccessToken(testSecret, user, "s1")
	if err != nil {
		t.Fatal(err)
	}
	uid, sid, err := middleware.ParseAccessToken(testSecret, good)
	if err != nil || uid != "u1" || sid != "s1" {
		t.Fatalf("valid token rejected: %v %v %v", uid, sid, err)
	}
	if _, _, err := middleware.ParseAccessToken([]byte("another-secret-another-secret-123456"), good); err == nil {
		t.Fatal("wrong secret accepted")
	}

	forge := func(m jwt.SigningMethod, claims jwt.MapClaims, key any) string {
		s, err := jwt.NewWithClaims(m, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	exp := time.Now().Add(time.Hour).Unix()
	tokens := map[string]string{
		"hs512":      forge(jwt.SigningMethodHS512, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": exp}, testSecret),
		"none":       forge(jwt.SigningMethodNone, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": exp}, jwt.UnsafeAllowNoneSignatureType),
		"no exp":     forge(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u1", "sid": "s1"}, testSecret),
		"expired":    forge(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(-time.Minute).Unix()}, testSecret),
		"no uid":     forge(jwt.SigningMethodHS256, jwt.MapClaims{"sid": "s1", "exp": exp}, testSecret),
		"no session": forge(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u1", "exp": exp}, testSecret),
	}
	for name, tok := range tokens {
		if _, _, err := middleware.ParseAccessToken(testSecret, tok); err == nil {
			t.Errorf("%s token must be rejected", name)
		}
	}
}

func TestServiceLifecycle(t *testing.T) {
	pool := testPool(t)
	svc := NewService(NewPGRepository(pool), testSecret)
	email := "svc-" + time.Now().Format("150405.000000") + "@itest.invalid"
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email) })

	tokens, user, err := svc.Register(context.Background(), "  "+strings.ToUpper(email)+" ", "Password123", "agent", "127.0.0.1")
	if err != nil || user.Email != email || tokens.AccessToken == "" {
		t.Fatalf("register: %v %+v", err, user)
	}
	if _, _, err := svc.Register(context.Background(), email, "Password123", "", ""); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
	if _, _, err := svc.Register(context.Background(), "not-an-email", "Password123", "", ""); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
	if _, _, err := svc.Login(context.Background(), email, "bad-password", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	// stored hash is bcrypt, never the password
	var hash string
	_ = pool.QueryRow(context.Background(), `SELECT password_hash FROM users WHERE id = $1`, user.ID).Scan(&hash)
	if !strings.HasPrefix(hash, "$2") || strings.Contains(hash, "Password123") {
		t.Fatalf("password must be stored as a bcrypt hash, got %q", hash)
	}

	// a disabled account can no longer log in nor use its tokens
	_, _ = pool.Exec(context.Background(), `UPDATE users SET disabled_at = NOW() WHERE id = $1`, user.ID)
	if _, _, err := svc.Login(context.Background(), email, "Password123", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("disabled user must not log in, got %v", err)
	}
	uid, sid, _ := middleware.ParseAccessToken(testSecret, tokens.AccessToken)
	if svc.ValidateSession(context.Background(), uid, sid) {
		t.Fatal("disabled user's session must be invalid")
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	pool, err := db.ConnectWithRetry(context.Background(), dsn, 10*time.Second, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	return pool
}
