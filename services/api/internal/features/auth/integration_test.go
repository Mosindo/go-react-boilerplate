package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestValidatePassword(t *testing.T) {
	ok := []string{"Password123", "motdepasse9", "héllo1234"}
	bad := []string{"", "short1", "onlyletters", "12345678", strings.Repeat("a1", 40)}
	for _, p := range ok {
		if err := validatePassword(p); err != nil {
			t.Errorf("%q should be accepted: %v", p, err)
		}
	}
	for _, p := range bad {
		if err := validatePassword(p); err == nil {
			t.Errorf("%q should be rejected", p)
		}
	}
}

func TestResetCodeAlphabetAvoidsAmbiguousCharacters(t *testing.T) {
	for i := 0; i < 200; i++ {
		code, err := randomCode(resetCodeLength)
		if err != nil || len(code) != resetCodeLength {
			t.Fatalf("bad code %q %v", code, err)
		}
		if strings.ContainsAny(code, "01OIL") {
			t.Fatalf("ambiguous character in %q", code)
		}
	}
}

func TestHashSecretIsStableAndEmptySafe(t *testing.T) {
	if hashSecret("abc") != hashSecret("abc") || hashSecret("abc") == hashSecret("abd") {
		t.Fatal("hashSecret must be deterministic and collision-free for different inputs")
	}
	if hashSecret("   ") != "" {
		t.Fatal("blank secrets hash to empty so they never match a stored token")
	}
}

func TestServiceRegisterLoginMeAndDelete(t *testing.T) {
	pool := setupTestDB(t)
	svc := NewService(NewPGRepository(pool), []byte("services-test-secret-0123456789abcdef"), nil, nil)
	email := "svc_" + time.Now().Format("150405.000000") + "@test.invalid"
	defer func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email) }()

	tokens, user, err := svc.Register(context.Background(), "  "+strings.ToUpper(email)+" ", "Password123", "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != email || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("unexpected registration result: %+v", user)
	}
	if _, _, err := svc.Register(context.Background(), email, "Password123", "", ""); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
	if _, _, err := svc.Login(context.Background(), email, "wrong", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if _, err := svc.Me(context.Background(), user.ID); err != nil {
		t.Fatalf("me: %v", err)
	}
	if err := svc.DeleteAccount(context.Background(), user.ID, "Nope123456"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("delete with wrong password: %v", err)
	}
	if err := svc.DeleteAccount(context.Background(), user.ID, "Password123"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Me(context.Background(), user.ID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound after deletion, got %v", err)
	}
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}
