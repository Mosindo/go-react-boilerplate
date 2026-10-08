package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/testutil"
)

type captureMailer struct{ lastBody string }

func (m *captureMailer) Send(_ context.Context, _, _, body string) error {
	m.lastBody = body
	return nil
}

var _ mailer.Mailer = (*captureMailer)(nil)

func newService(t *testing.T) (*Service, *captureMailer) {
	t.Helper()
	pool := testutil.NewPool(t)
	m := &captureMailer{}
	return NewService(NewPGRepository(pool), []byte("0123456789abcdef0123456789abcdef"), m, nil), m
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	tokens, user, err := svc.Register(ctx, "  Alice@Example.COM ", "Passw0rd!x", "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "alice@example.com" || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("unexpected register result: %+v %+v", user, tokens)
	}
	if _, _, err := svc.Register(ctx, "alice@example.com", "Passw0rd!x", "", ""); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
	if _, _, err := svc.Login(ctx, "alice@example.com", "wrong-password1", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if _, _, err := svc.Login(ctx, "nobody@example.com", "whatever123", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email must look like bad password, got %v", err)
	}

	login, _, err := svc.Login(ctx, "ALICE@example.com", "Passw0rd!x", "", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	rotated, _, err := svc.Refresh(ctx, login.RefreshToken, "", "")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if rotated.RefreshToken == login.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	if _, _, err := svc.Refresh(ctx, login.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("old refresh token must be rejected, got %v", err)
	}
	if err := svc.Logout(ctx, rotated.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, rotated.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("logged-out token must be rejected, got %v", err)
	}
}

func TestPasswordRecovery(t *testing.T) {
	svc, mail := newService(t)
	ctx := context.Background()
	tokens, _, err := svc.Register(ctx, "bob@example.com", "Passw0rd!x", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.ForgotPassword(ctx, "ghost@example.com"); err != nil || mail.lastBody != "" {
		t.Fatalf("unknown email must be silent and send nothing: err=%v body=%q", err, mail.lastBody)
	}
	if err := svc.ForgotPassword(ctx, "bob@example.com"); err != nil {
		t.Fatal(err)
	}
	code := extractCode(t, mail.lastBody)

	if err := svc.ResetPassword(ctx, "bob@example.com", "WRONGCODE", "NewPassw0rd!"); err == nil {
		t.Fatal("wrong code must fail")
	}
	if err := svc.ResetPassword(ctx, "bob@example.com", strings.ToLower(code), "short"); err == nil {
		t.Fatal("weak new password must fail")
	}
	if err := svc.ResetPassword(ctx, "bob@example.com", strings.ToLower(code), "NewPassw0rd!"); err != nil {
		t.Fatalf("reset with valid code (case-insensitive): %v", err)
	}
	if err := svc.ResetPassword(ctx, "bob@example.com", code, "Another1Pass!"); err == nil {
		t.Fatal("a code must be single use")
	}
	if _, _, err := svc.Refresh(ctx, tokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatal("password reset must revoke existing sessions")
	}
	if _, _, err := svc.Login(ctx, "bob@example.com", "NewPassw0rd!", "", ""); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestResetCodeLockoutAfterTooManyAttempts(t *testing.T) {
	svc, mail := newService(t)
	ctx := context.Background()
	if _, _, err := svc.Register(ctx, "carol@example.com", "Passw0rd!x", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.ForgotPassword(ctx, "carol@example.com"); err != nil {
		t.Fatal(err)
	}
	code := extractCode(t, mail.lastBody)
	for i := 0; i < resetMaxTries; i++ {
		_ = svc.ResetPassword(ctx, "carol@example.com", "AAAAAAAA", "NewPassw0rd!")
	}
	if err := svc.ResetPassword(ctx, "carol@example.com", code, "NewPassw0rd!"); err == nil {
		t.Fatal("the correct code must be refused once attempts are exhausted")
	}
}

func TestValidation(t *testing.T) {
	for _, bad := range []string{"", "no-at-sign", "a@b", "a b@example.com", "<x>@example.com"} {
		if _, err := validateEmail(bad); err == nil {
			t.Errorf("email %q should be rejected", bad)
		}
	}
	for _, bad := range []string{"short1!", "alllettersnodigit", "12345678901"} {
		if err := validatePassword(bad); err == nil {
			t.Errorf("password %q should be rejected", bad)
		}
	}
	if err := validatePassword(strings.Repeat("a1", 37)); err == nil {
		t.Error("passwords over 72 bytes must be rejected (bcrypt limit)")
	}
	if err := validatePassword("Correct-Horse-9"); err != nil {
		t.Errorf("valid password rejected: %v", err)
	}
}

func extractCode(t *testing.T, body string) string {
	t.Helper()
	const marker = "réinitialisation : "
	i := strings.Index(body, marker)
	if i < 0 {
		t.Fatalf("no code in mail body %q", body)
	}
	return body[i+len(marker) : i+len(marker)+resetCodeLen]
}
