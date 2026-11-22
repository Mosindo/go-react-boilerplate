package auth

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthServiceRegisterLoginMe(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	authRepo := NewPGRepository(pool)
	svc := NewService(authRepo, []byte("services-test-secret"))
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

func TestPasswordResetFlow(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	mail := &mailer.Memory{}
	svc := NewService(NewPGRepository(pool), []byte("services-test-secret"), WithMailer(mail))
	email := uniqueEmail("reset")
	defer cleanupUsers(t, pool, []string{email})

	_, user, err := svc.Register(context.Background(), email, "Password123", "", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	tokens, _, err := svc.Login(context.Background(), email, "Password123", "", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	// Unknown e-mails get the same silent success (no account enumeration).
	if err := svc.ForgotPassword(context.Background(), "nobody_"+email); err != nil {
		t.Fatalf("forgot unknown: %v", err)
	}
	if len(mail.Sent) != 0 {
		t.Fatalf("no mail must be sent for unknown accounts")
	}

	if err := svc.ForgotPassword(context.Background(), strings.ToUpper(email)); err != nil {
		t.Fatalf("forgot: %v", err)
	}
	msg, ok := mail.Last()
	if !ok || msg.To != email {
		t.Fatalf("expected a reset mail to %s, got %+v", email, msg)
	}
	code := regexp.MustCompile(`\b\d{6}\b`).FindString(msg.Body)
	if code == "" {
		t.Fatalf("reset mail must contain a 6-digit code: %q", msg.Body)
	}

	wrong := "000000"
	if wrong == code {
		wrong = "111111"
	}
	if err := svc.ResetPassword(context.Background(), email, wrong, "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("wrong code expected ErrInvalidResetCode, got %v", err)
	}
	if err := svc.ResetPassword(context.Background(), email, code, "weak"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("weak password expected ErrWeakPassword, got %v", err)
	}
	if err := svc.ResetPassword(context.Background(), email, code, "NewPassword456"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), email, code, "OtherPassword789"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("codes are single-use, got %v", err)
	}
	if _, _, err := svc.Refresh(context.Background(), tokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("reset must revoke existing sessions, got %v", err)
	}
	if _, _, err := svc.Login(context.Background(), email, "NewPassword456", "", ""); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	_ = user
}

func TestResetCodeLocksAfterTooManyAttempts(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	mail := &mailer.Memory{}
	svc := NewService(NewPGRepository(pool), []byte("services-test-secret"), WithMailer(mail))
	email := uniqueEmail("reset_lock")
	defer cleanupUsers(t, pool, []string{email})
	if _, _, err := svc.Register(context.Background(), email, "Password123", "", ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.ForgotPassword(context.Background(), email); err != nil {
		t.Fatalf("forgot: %v", err)
	}
	msg, _ := mail.Last()
	code := regexp.MustCompile(`\b\d{6}\b`).FindString(msg.Body)
	wrong := "000000"
	if wrong == code {
		wrong = "111111"
	}
	for i := 0; i < maxResetCodeAttempts; i++ {
		_ = svc.ResetPassword(context.Background(), email, wrong, "NewPassword456")
	}
	if err := svc.ResetPassword(context.Background(), email, code, "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("code must be locked after %d failures, got %v", maxResetCodeAttempts, err)
	}
}

func TestChangePasswordAndDeleteAccount(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	var deleted DeletedAccount
	svc := NewService(NewPGRepository(pool), []byte("services-test-secret"), WithDeletionHook(func(d DeletedAccount) { deleted = d }))
	email := uniqueEmail("account")
	defer cleanupUsers(t, pool, []string{email})

	first, user, err := svc.Register(context.Background(), email, "Password123", "", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	second, _, err := svc.Login(context.Background(), email, "Password123", "", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	firstSession := sessionIDFromRefresh(t, svc, first.RefreshToken)

	if err := svc.ChangePassword(context.Background(), user.ID, firstSession, "Wrong1234", "Password456"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
	if err := svc.ChangePassword(context.Background(), user.ID, firstSession, "Password123", "Password456"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if active, _ := svc.SessionActive(context.Background(), firstSession, user.ID); !active {
		t.Fatalf("current session must survive a password change")
	}
	if _, _, err := svc.Refresh(context.Background(), second.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("other sessions must be revoked, got %v", err)
	}

	if err := svc.DeleteAccount(context.Background(), user.ID, "Password123"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("deletion requires the current password, got %v", err)
	}
	if err := svc.DeleteAccount(context.Background(), user.ID, "Password456"); err != nil {
		t.Fatalf("delete account: %v", err)
	}
	if deleted.UserID != user.ID {
		t.Fatalf("deletion hook not called")
	}
	if _, err := svc.Me(context.Background(), user.ID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("deleted account must be gone, got %v", err)
	}
	var sessions int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM sessions WHERE user_id = $1`, user.ID).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("sessions must be erased with the account (count=%d err=%v)", sessions, err)
	}
}

func sessionIDFromRefresh(t *testing.T, svc *Service, refreshToken string) string {
	t.Helper()
	session, err := svc.repo.GetActiveSessionByTokenHash(context.Background(), hashRefreshToken(refreshToken))
	if err != nil {
		t.Fatalf("lookup session: %v", err)
	}
	return session.ID
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
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000000") + "@lueur.test"
}

func cleanupUsers(t *testing.T, pool *pgxpool.Pool, emails []string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, emails)
	if err != nil {
		t.Fatalf("cleanup users: %v", err)
	}
}
