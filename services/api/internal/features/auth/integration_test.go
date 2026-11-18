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

func TestAuthServiceRegisterLoginMe(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	authRepo := NewPGRepository(pool)
	svc := NewService(authRepo, []byte("services-test-secret-0123456789abcdef"), &captureMailer{})
	email := uniqueEmail("authsvc")
	password := "Password123"
	defer cleanupUsers(t, pool, []string{email})

	tokens, user, err := svc.Register(context.Background(), "  "+strings.ToUpper(email)+"  ", password, "auth-test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected non-empty access and refresh tokens")
	}
	if user.Email != email {
		t.Fatalf("expected normalized email %q, got %q", email, user.Email)
	}
	if user.OrganizationID == "" {
		t.Fatalf("expected organization id on registered user")
	}

	_, _, err = svc.Register(context.Background(), email, password, "", "")
	if !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}

	_, _, err = svc.Login(context.Background(), email, "wrong-password", "", "")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	loginTokens, loggedIn, err := svc.Login(context.Background(), email, password, "auth-test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginTokens.AccessToken == "" || loginTokens.RefreshToken == "" {
		t.Fatalf("expected non-empty login tokens")
	}
	if loggedIn.ID != user.ID {
		t.Fatalf("expected login user id %s, got %s", user.ID, loggedIn.ID)
	}
	if loggedIn.OrganizationID != user.OrganizationID {
		t.Fatalf("expected login user organization id %s, got %s", user.OrganizationID, loggedIn.OrganizationID)
	}

	refreshedTokens, refreshedUser, err := svc.Refresh(context.Background(), loginTokens.RefreshToken, "auth-test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshedTokens.AccessToken == "" || refreshedTokens.RefreshToken == "" {
		t.Fatalf("expected non-empty refreshed tokens")
	}
	if refreshedTokens.RefreshToken == loginTokens.RefreshToken {
		t.Fatalf("expected rotated refresh token")
	}
	if refreshedUser.OrganizationID != user.OrganizationID {
		t.Fatalf("expected refreshed user organization id %s, got %s", user.OrganizationID, refreshedUser.OrganizationID)
	}

	if err := svc.Logout(context.Background(), refreshedTokens.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}

	_, _, err = svc.Refresh(context.Background(), refreshedTokens.RefreshToken, "auth-test-agent", "127.0.0.1")
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken after logout, got %v", err)
	}

	meUser, err := svc.Me(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("me existing user: %v", err)
	}
	if meUser.Email != email {
		t.Fatalf("expected me email %q, got %q", email, meUser.Email)
	}
	if meUser.OrganizationID != user.OrganizationID {
		t.Fatalf("expected me organization id %s, got %s", user.OrganizationID, meUser.OrganizationID)
	}

	_, err = svc.Me(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

type captureMailer struct {
	body string
}

func (m *captureMailer) Send(_ context.Context, _, _, body string) error {
	m.body = body
	return nil
}

func TestPasswordResetFlow(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	mailer := &captureMailer{}
	svc := NewService(NewPGRepository(pool), []byte("services-test-secret-0123456789abcdef"), mailer)
	email := uniqueEmail("reset")
	defer cleanupUsers(t, pool, []string{email})

	tokens, user, err := svc.Register(context.Background(), email, "Password123", "", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.RequestPasswordReset(context.Background(), "unknown_"+email); err != nil {
		t.Fatalf("unknown e-mail must not error: %v", err)
	}
	if mailer.body != "" {
		t.Fatal("no mail must be sent for unknown e-mail")
	}

	if err := svc.RequestPasswordReset(context.Background(), email); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	code := mailer.body[strings.Index(mailer.body, ": ")+2:][:6]

	wrong := "000000"
	if code == wrong {
		wrong = "999999"
	}
	if err := svc.ResetPassword(context.Background(), email, wrong, "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("expected wrong code rejection, got %v", err)
	}
	if err := svc.ResetPassword(context.Background(), email, code, "weak"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected weak password rejection, got %v", err)
	}
	if err := svc.ResetPassword(context.Background(), email, code, "NewPassword456"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), email, code, "NewPassword789"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("code must be single use, got %v", err)
	}
	if _, _, err := svc.Refresh(context.Background(), tokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("existing sessions must be revoked after reset, got %v", err)
	}
	if _, _, err := svc.Login(context.Background(), email, "NewPassword456", "", ""); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	_ = user
}

func TestResetCodeLocksAfterTooManyAttempts(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	mailer := &captureMailer{}
	svc := NewService(NewPGRepository(pool), []byte("services-test-secret-0123456789abcdef"), mailer)
	email := uniqueEmail("resetlock")
	defer cleanupUsers(t, pool, []string{email})
	if _, _, err := svc.Register(context.Background(), email, "Password123", "", ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.RequestPasswordReset(context.Background(), email); err != nil {
		t.Fatal(err)
	}
	code := mailer.body[strings.Index(mailer.body, ": ")+2:][:6]
	wrong := "111111"
	if code == wrong {
		wrong = "222222"
	}
	for i := 0; i < maxResetAttempts; i++ {
		_ = svc.ResetPassword(context.Background(), email, wrong, "NewPassword456")
	}
	if err := svc.ResetPassword(context.Background(), email, code, "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("code must be locked after too many attempts, got %v", err)
	}
}

func TestValidatePassword(t *testing.T) {
	for _, bad := range []string{"short1", "onlyletters", "12345678", strings.Repeat("a1", 40)} {
		if err := ValidatePassword(bad); !errors.Is(err, ErrWeakPassword) {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
	if err := ValidatePassword("Password123"); err != nil {
		t.Fatalf("expected valid password: %v", err)
	}
}

func setupFeaturesTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("skip features integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}

	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("skip features integration tests: postgres unavailable (%v)", err)
	}

	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}

	return pool
}

func uniqueEmail(prefix string) string {
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000000") + "@boilerplate.test"
}

func cleanupUsers(t *testing.T, pool *pgxpool.Pool, emails []string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, emails)
	if err != nil {
		t.Fatalf("cleanup users: %v", err)
	}
}

func TestValidEmail(t *testing.T) {
	for _, ok := range []string{"a@b.co", "lea.martin+tag@mail.example.fr"} {
		if !ValidEmail(ok) {
			t.Fatalf("expected %q to be valid", ok)
		}
	}
	for _, bad := range []string{"", "plain", "a@b", "Léa <lea@x.fr>", "a b@c.fr", "@x.fr"} {
		if ValidEmail(bad) {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}
