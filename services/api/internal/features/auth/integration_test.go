package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db/dbtest"
)

type captureMailer struct {
	mu   sync.Mutex
	body string
	to   string
}

func (m *captureMailer) Send(_ context.Context, to, _ string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.to, m.body = to, body
	return nil
}

// code waits for the asynchronously sent email.
func (m *captureMailer) code() string {
	for i := 0; i < 100; i++ {
		m.mu.Lock()
		body := m.body
		m.mu.Unlock()
		if match := regexp.MustCompile(`: ([A-Z2-9]{10})\b`).FindStringSubmatch(body); match != nil {
			return match[1]
		}
		time.Sleep(20 * time.Millisecond)
	}
	return ""
}

func newTestService(t *testing.T) (*Service, *captureMailer) {
	t.Helper()
	pool := dbtest.NewPool(t)
	mail := &captureMailer{}
	return NewService(NewPGRepository(pool), []byte("services-test-secret-0123456789ab"), mail, "Test"), mail
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	email := "ada@example.test"
	password := "Password123"

	tokens, user, err := svc.Register(ctx, "  "+strings.ToUpper(email)+" ", password, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
	if user.Email != email {
		t.Fatalf("email must be trimmed and lowercased, got %q", user.Email)
	}

	if _, _, err := svc.Register(ctx, email, password, "", ""); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("duplicate email: got %v", err)
	}
	if _, _, err := svc.Register(ctx, "short@example.test", "short", "", ""); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("short password: got %v", err)
	}
	if _, _, err := svc.Register(ctx, "long@example.test", strings.Repeat("a", 73), "", ""); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("over-long password: got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "wrong-password", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: got %v", err)
	}
	if _, _, err := svc.Login(ctx, "ghost@example.test", password, "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user must look like a wrong password: got %v", err)
	}

	login, _, err := svc.Login(ctx, email, password, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	rotated, _, err := svc.Refresh(ctx, login.RefreshToken, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if rotated.RefreshToken == login.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	if _, _, err := svc.Refresh(ctx, login.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("old refresh token must be dead: got %v", err)
	}

	if err := svc.Logout(ctx, rotated.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, rotated.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("refresh after logout: got %v", err)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	svc, mail := newTestService(t)
	ctx := context.Background()
	email := "grace@example.test"

	tokens, _, err := svc.Register(ctx, email, "OldPassword1", "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Unknown address: silent success, nothing sent.
	if err := svc.RequestPasswordReset(ctx, "nobody@example.test"); err != nil {
		t.Fatalf("unknown email must not error: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	mail.mu.Lock()
	sent := mail.body
	mail.mu.Unlock()
	if sent != "" {
		t.Fatal("no mail may be sent for unknown addresses")
	}

	if err := svc.RequestPasswordReset(ctx, email); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	code := mail.code()

	if err := svc.ResetPassword(ctx, email, "WRONGCODE2", "NewPassword1"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("wrong code: got %v", err)
	}
	if err := svc.ResetPassword(ctx, email, code, "short"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("weak new password: got %v", err)
	}
	if err := svc.ResetPassword(ctx, email, strings.ToLower(code), "NewPassword1"); err != nil {
		t.Fatalf("reset with correct code (case-insensitive): %v", err)
	}
	if err := svc.ResetPassword(ctx, email, code, "AnotherPass1"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("code must be single use: got %v", err)
	}

	if _, _, err := svc.Login(ctx, email, "OldPassword1", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password must stop working: got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "NewPassword1", "", ""); err != nil {
		t.Fatalf("new password must work: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, tokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("reset must revoke existing sessions: got %v", err)
	}
}

func TestResetCodeLocksAfterTooManyAttempts(t *testing.T) {
	svc, mail := newTestService(t)
	ctx := context.Background()
	email := "lock@example.test"
	if _, _, err := svc.Register(ctx, email, "Password123", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.RequestPasswordReset(ctx, email); err != nil {
		t.Fatal(err)
	}
	code := mail.code()

	for i := 0; i < resetMaxAttempts; i++ {
		_ = svc.ResetPassword(ctx, email, "WRONGCODE2", "NewPassword1")
	}
	if err := svc.ResetPassword(ctx, email, code, "NewPassword1"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("code must be locked after %d bad attempts: got %v", resetMaxAttempts, err)
	}
}
