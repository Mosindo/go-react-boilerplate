package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testPassword = "Password123"

type captureMailer struct {
	mu       sync.Mutex
	messages []string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, to+"\n"+body)
	return nil
}

func (m *captureMailer) lastCode(t *testing.T) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		t.Fatal("no email was sent")
	}
	match := regexp.MustCompile(`\b(\d{6})\b`).FindStringSubmatch(m.messages[len(m.messages)-1])
	if match == nil {
		t.Fatal("no 6-digit code in email")
	}
	return match[1]
}

func (m *captureMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

type recordingHooks struct{ deleted []string }

func (h *recordingHooks) BeforeDelete(_ context.Context, userID string) (func(), error) {
	return func() { h.deleted = append(h.deleted, userID) }, nil
}

func newTestService(t *testing.T) (*Service, *pgxpool.Pool, *captureMailer) {
	t.Helper()
	pool := setupTestDB(t)
	mail := &captureMailer{}
	svc := NewService(NewPGRepository(pool), []byte("auth-tests-signing-key-0123456789"))
	svc.SetMailer(mail)
	t.Cleanup(pool.Close)
	return svc, pool, mail
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	svc, pool, _ := newTestService(t)
	ctx := context.Background()
	email := uniqueEmail("flow")
	t.Cleanup(func() { cleanupUsers(t, pool, email) })

	tokens, user, err := svc.Register(ctx, "  "+strings.ToUpper(email)+"  ", testPassword, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != email {
		t.Fatalf("email should be trimmed and lowercased, got %q", user.Email)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected both tokens")
	}

	if _, _, err := svc.Register(ctx, email, testPassword, "", ""); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("duplicate register: expected ErrEmailExists, got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "wrong-password", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("bad password: expected ErrInvalidCredentials, got %v", err)
	}
	if _, _, err := svc.Login(ctx, "nobody-"+email, testPassword, "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email: expected ErrInvalidCredentials, got %v", err)
	}

	loginTokens, loggedIn, err := svc.Login(ctx, email, testPassword, "", "")
	if err != nil || loggedIn.ID != user.ID {
		t.Fatalf("login: %v", err)
	}

	refreshed, _, err := svc.Refresh(ctx, loginTokens.RefreshToken, "", "")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.RefreshToken == loginTokens.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	if _, _, err := svc.Refresh(ctx, loginTokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("reusing a rotated refresh token must fail, got %v", err)
	}

	if err := svc.Logout(ctx, refreshed.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, refreshed.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("refresh after logout must fail, got %v", err)
	}
}

func TestPasswordRules(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()

	for name, password := range map[string]string{
		"too short":      "short",
		"over 72 bytes":  strings.Repeat("é", 40), // 80 bytes, 40 runes
		"empty":          "",
		"just under min": "1234567",
	} {
		if _, _, err := svc.Register(ctx, uniqueEmail("weak"), password, "", ""); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("%s: expected ErrWeakPassword, got %v", name, err)
		}
	}
}

func TestSessionActiveUntilLogoutOrDeletion(t *testing.T) {
	svc, pool, _ := newTestService(t)
	ctx := context.Background()
	email := uniqueEmail("session")
	t.Cleanup(func() { cleanupUsers(t, pool, email) })

	tokens, user, err := svc.Register(ctx, email, testPassword, "", "")
	if err != nil {
		t.Fatal(err)
	}
	var sessionID string
	if err := pool.QueryRow(ctx, `SELECT id FROM sessions WHERE user_id = $1`, user.ID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	if active, err := svc.IsSessionActive(ctx, user.ID, sessionID); err != nil || !active {
		t.Fatalf("session should be active: %v %v", active, err)
	}
	if err := svc.Logout(ctx, tokens.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if active, _ := svc.IsSessionActive(ctx, user.ID, sessionID); active {
		t.Fatal("session must be inactive after logout")
	}
}

func TestPasswordResetFlow(t *testing.T) {
	svc, pool, mail := newTestService(t)
	ctx := context.Background()
	email := uniqueEmail("reset")
	t.Cleanup(func() { cleanupUsers(t, pool, email) })

	oldTokens, _, err := svc.Register(ctx, email, testPassword, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Unknown addresses succeed silently and send nothing.
	if err := svc.RequestPasswordReset(ctx, "ghost-"+email); err != nil {
		t.Fatalf("unknown email must not error: %v", err)
	}
	if mail.count() != 0 {
		t.Fatal("no email should be sent for unknown accounts")
	}

	if err := svc.RequestPasswordReset(ctx, email); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	code := mail.lastCode(t)

	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	if err := svc.ResetPassword(ctx, email, wrong, "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("wrong code must be rejected, got %v", err)
	}
	if err := svc.ResetPassword(ctx, email, code, "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("weak new password must be rejected, got %v", err)
	}
	if err := svc.ResetPassword(ctx, email, code, "NewPassword456"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := svc.ResetPassword(ctx, email, code, "AnotherPass789"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("a code can only be used once, got %v", err)
	}

	if _, _, err := svc.Login(ctx, email, testPassword, "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password must stop working, got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "NewPassword456", "", ""); err != nil {
		t.Fatalf("new password must work: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, oldTokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("existing sessions must be revoked by a reset, got %v", err)
	}
}

func TestPasswordResetLocksAfterTooManyGuesses(t *testing.T) {
	svc, pool, mail := newTestService(t)
	ctx := context.Background()
	email := uniqueEmail("lock")
	t.Cleanup(func() { cleanupUsers(t, pool, email) })

	if _, _, err := svc.Register(ctx, email, testPassword, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.RequestPasswordReset(ctx, email); err != nil {
		t.Fatal(err)
	}
	code := mail.lastCode(t)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < maxResetAttempts; i++ {
		if err := svc.ResetPassword(ctx, email, wrong, "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
			t.Fatalf("guess %d: expected ErrInvalidResetCode, got %v", i, err)
		}
	}
	if err := svc.ResetPassword(ctx, email, code, "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("the right code must be refused once attempts are exhausted, got %v", err)
	}
}

func TestChangePasswordAndDeleteAccount(t *testing.T) {
	svc, pool, _ := newTestService(t)
	hooks := &recordingHooks{}
	svc.SetAccountHooks(hooks)
	ctx := context.Background()
	email := uniqueEmail("delete")
	t.Cleanup(func() { cleanupUsers(t, pool, email) })

	tokens, user, err := svc.Register(ctx, email, testPassword, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.ChangePassword(ctx, user.ID, "not-my-password", "NewPassword456"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
	if err := svc.ChangePassword(ctx, user.ID, testPassword, "NewPassword456"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, tokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("changing the password must sign out other sessions, got %v", err)
	}

	if err := svc.DeleteAccount(ctx, user.ID, "wrong"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("deleting requires the password, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, user.ID, "NewPassword456"); err != nil {
		t.Fatalf("delete account: %v", err)
	}
	if len(hooks.deleted) != 1 || hooks.deleted[0] != user.ID {
		t.Fatalf("cleanup hook should run once after deletion, got %v", hooks.deleted)
	}
	if _, _, err := svc.Login(ctx, email, "NewPassword456", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("a deleted account cannot sign in, got %v", err)
	}
	var sessions int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessions WHERE user_id = $1`, user.ID).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("sessions must be deleted with the account: %d %v", sessions, err)
	}
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}

	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("skip integration tests: postgres unavailable (%v)", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	return pool
}

var emailCounter struct {
	sync.Mutex
	n int
}

func uniqueEmail(prefix string) string {
	emailCounter.Lock()
	defer emailCounter.Unlock()
	emailCounter.n++
	return fmt.Sprintf("%s_%d_%d@dating.test", prefix, time.Now().UnixNano(), emailCounter.n)
}

func cleanupUsers(t *testing.T, pool *pgxpool.Pool, emails ...string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, emails); err != nil {
		t.Errorf("cleanup users: %v", err)
	}
}
