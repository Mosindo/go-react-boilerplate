package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/testutil"
)

type captureMailer struct{ last mailer.Message }

func (m *captureMailer) Send(_ context.Context, msg mailer.Message) error {
	m.last = msg
	return nil
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%d@auth.test", prefix, time.Now().UnixNano())
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	pool := testutil.DB(t)
	svc := NewService(NewPGRepository(pool), []byte("0123456789abcdef0123456789abcdef"))
	email := uniqueEmail("flow")
	ctx := context.Background()

	tokens, user, err := svc.Register(ctx, "  "+strings.ToUpper(email)+" ", "Password123", "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) })
	if user.Email != email {
		t.Fatalf("email must be normalized, got %q", user.Email)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
	if _, _, err := svc.Register(ctx, email, "Password123", "", ""); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("duplicate email: got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "wrong-password", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: got %v", err)
	}
	if _, _, err := svc.Login(ctx, "nobody@auth.test", "Password123", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email: got %v", err)
	}
	login, _, err := svc.Login(ctx, email, "Password123", "", "")
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
		t.Fatalf("refresh after logout: got %v", err)
	}
}

func TestPasswordRecoveryAndChange(t *testing.T) {
	pool := testutil.DB(t)
	mail := &captureMailer{}
	svc := NewService(NewPGRepository(pool), []byte("0123456789abcdef0123456789abcdef")).WithMailer(mail)
	email := uniqueEmail("reset")
	ctx := context.Background()

	tokens, user, err := svc.Register(ctx, email, "Password123", "", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) })

	// Unknown account: silent success, nothing sent.
	if err := svc.ForgotPassword(ctx, "ghost@auth.test"); err != nil || mail.last.To != "" {
		t.Fatalf("unknown email must be silent, err=%v mail=%v", err, mail.last)
	}
	if err := svc.ForgotPassword(ctx, email); err != nil {
		t.Fatal(err)
	}
	code := strings.TrimSuffix(strings.TrimPrefix(mail.last.Body, "Your reset code is "), ". It expires in 1 hour. If you did not ask for it, ignore this message.")
	if len(code) != resetCodeLength {
		t.Fatalf("unexpected code %q", code)
	}
	if err := svc.ResetPassword(ctx, email, "WRONGCODE1", "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("wrong code: got %v", err)
	}
	if err := svc.ResetPassword(ctx, "ghost@auth.test", code, "NewPassword456"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("code must be bound to its email: got %v", err)
	}
	if err := svc.ResetPassword(ctx, email, strings.ToLower(code), "NewPassword456"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := svc.ResetPassword(ctx, email, code, "Another789xx"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("code must be single use: got %v", err)
	}
	if _, _, err := svc.Refresh(ctx, tokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("sessions must be revoked after reset: got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "Password123", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("old password must stop working")
	}
	if _, _, err := svc.Login(ctx, email, "NewPassword456", "", ""); err != nil {
		t.Fatalf("new password: %v", err)
	}

	if err := svc.ChangePassword(ctx, user.ID, "bad", "Whatever123"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("change with wrong current password: got %v", err)
	}
	if err := svc.ChangePassword(ctx, user.ID, "NewPassword456", "Final12345"); err != nil {
		t.Fatalf("change: %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "Final12345", "", ""); err != nil {
		t.Fatalf("login after change: %v", err)
	}
}

func TestDeleteAccountRequiresPasswordAndRunsHooks(t *testing.T) {
	pool := testutil.DB(t)
	hook := &countingHook{}
	svc := NewService(NewPGRepository(pool), []byte("0123456789abcdef0123456789abcdef")).WithDeletionHooks(hook)
	ctx := context.Background()
	email := uniqueEmail("del")
	_, user, err := svc.Register(ctx, email, "Password123", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteAccount(ctx, user.ID, "nope"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("got %v", err)
	}
	if hook.cleaned != 0 {
		t.Fatal("cleanup must not run when the password is wrong")
	}
	if err := svc.DeleteAccount(ctx, user.ID, "Password123"); err != nil {
		t.Fatal(err)
	}
	if hook.prepared != 1 || hook.cleaned != 1 {
		t.Fatalf("hook calls: %+v", hook)
	}
	if _, err := svc.Me(ctx, user.ID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("user must be gone, got %v", err)
	}
}

type countingHook struct{ prepared, cleaned int }

func (h *countingHook) PrepareUserDeletion(context.Context, string) (func(), error) {
	h.prepared++
	return func() { h.cleaned++ }, nil
}
