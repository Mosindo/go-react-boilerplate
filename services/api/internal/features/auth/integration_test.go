package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/testutil"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	pool := testutil.Pool(t)
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return NewService(NewPGRepository(pool), []byte("auth-service-test-secret-0123456789"), mail.LogMailer{}, store)
}

func code(err error) string {
	if be, ok := perr.As(err); ok {
		return be.Code
	}
	return ""
}

func TestServiceRegisterLoginMe(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	email := testutil.Email("authsvc")
	req := RegisterRequest{Email: "  " + strings.ToUpper(email) + "  ", Password: "Password123", BirthDate: "1990-05-20"}

	tokens, user, err := svc.Register(ctx, req, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
	if user.Email != email {
		t.Fatalf("email must be normalized: %q", user.Email)
	}

	if _, _, err := svc.Register(ctx, RegisterRequest{Email: email, Password: "Password123", BirthDate: "1990-05-20"}, "", ""); code(err) != "email_taken" {
		t.Fatalf("expected email_taken, got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "nope-nope", "", ""); code(err) != "invalid_credentials" {
		t.Fatalf("expected invalid_credentials, got %v", err)
	}
	if _, _, err := svc.Login(ctx, "ghost@test.alma.invalid", "Password123", "", ""); code(err) != "invalid_credentials" {
		t.Fatalf("unknown email must look like a wrong password, got %v", err)
	}
	_, logged, err := svc.Login(ctx, email, "Password123", "", "")
	if err != nil || logged.ID != user.ID {
		t.Fatalf("login: %v", err)
	}
	me, err := svc.Me(ctx, user.ID)
	if err != nil || me.ID != user.ID {
		t.Fatalf("me: %v", err)
	}
}

func TestChangePasswordRevokesOtherSessions(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	email := testutil.Email("chgpw")
	tokens, user, err := svc.Register(ctx, RegisterRequest{Email: email, Password: "Password123", BirthDate: "1990-05-20"}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	other, _, err := svc.Login(ctx, email, "Password123", "", "")
	if err != nil {
		t.Fatal(err)
	}
	session, err := svc.repo.GetActiveSessionByTokenHash(ctx, hashToken(tokens.RefreshToken))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangePassword(ctx, user.ID, session.ID, "wrong-password", "BrandNew123"); code(err) != "wrong_password" {
		t.Fatalf("wrong current password: %v", err)
	}
	if err := svc.ChangePassword(ctx, user.ID, session.ID, "Password123", "BrandNew123"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Refresh(ctx, other.RefreshToken, "", ""); code(err) != "invalid_refresh_token" {
		t.Fatalf("other sessions must be revoked, got %v", err)
	}
	if _, _, err := svc.Refresh(ctx, tokens.RefreshToken, "", ""); err != nil {
		t.Fatalf("current session must survive: %v", err)
	}
}

func TestExpiredResetCodeIsRejected(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	_, user, err := svc.Register(ctx, RegisterRequest{Email: testutil.Email("expired"), Password: "Password123", BirthDate: "1990-05-20"}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.ToUpper(fmt.Sprintf("%016x", time.Now().UnixNano())) // unique per run: the test DB is shared
	display := raw[0:4] + "-" + raw[4:8] + "-" + raw[8:12] + "-" + raw[12:16]
	if err := svc.repo.CreatePasswordReset(ctx, user.ID, hashToken(raw), time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetPassword(ctx, display, "Password456"); code(err) != "invalid_code" {
		t.Fatalf("expired code must be rejected, got %v", err)
	}
	if !errors.Is(svc.repo.ConsumePasswordReset(ctx, hashToken(raw), "x", time.Now()), ErrRepositoryResetInvalid) {
		t.Fatal("expired code must not be consumable")
	}
}
