package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"example.com/api/internal/platform/authtoken"
	"example.com/api/internal/testutil"
)

const testSecret = "integration-secret-0123456789abcdef"

type recordingCleaner struct {
	prepared  []string
	finalized []string
}

func (c *recordingCleaner) PrepareAccountDeletion(_ context.Context, userID string) (func(context.Context), error) {
	c.prepared = append(c.prepared, userID)
	return func(context.Context) { c.finalized = append(c.finalized, userID) }, nil
}

func newTestService(t *testing.T) (*Service, *testutil.CaptureMailer, *recordingCleaner, *PGRepository) {
	pool := testutil.DB(t)
	repo := NewPGRepository(pool)
	mail := &testutil.CaptureMailer{}
	cleaner := &recordingCleaner{}
	return NewService(repo, authtoken.NewManager([]byte(testSecret)), mail, cleaner), mail, cleaner, repo
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	svc, _, _, repo := newTestService(t)
	ctx := context.Background()
	email := testutil.Email(t, repo.dbPool, "auth")

	tokens, user, err := svc.Register(ctx, "  "+strings.ToUpper(email)+" ", "Password123", "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != email || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("unexpected register result: %+v", user)
	}

	var profileRows, prefRows int
	_ = repo.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM profiles WHERE user_id = $1`, user.ID).Scan(&profileRows)
	_ = repo.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM preferences WHERE user_id = $1`, user.ID).Scan(&prefRows)
	if profileRows != 1 || prefRows != 1 {
		t.Fatalf("registration must create profile and preferences rows (%d, %d)", profileRows, prefRows)
	}

	if _, _, err := svc.Register(ctx, email, "Password123", "", ""); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
	if _, _, err := svc.Register(ctx, "weak_"+email, "weak", "", ""); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "WrongPass1", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if _, _, err := svc.Login(ctx, "nobody@integration.test", "Password123", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email must look like invalid credentials, got %v", err)
	}

	login, _, err := svc.Login(ctx, email, "Password123", "agent", "127.0.0.1")
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
		t.Fatalf("reusing a rotated refresh token must fail, got %v", err)
	}

	claims, err := authtoken.NewManager([]byte(testSecret)).Parse(rotated.AccessToken, authtoken.TypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if active, _ := svc.SessionActive(ctx, claims.SessionID, user.ID); !active {
		t.Fatal("session should be active")
	}
	if err := svc.Logout(ctx, rotated.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if active, _ := svc.SessionActive(ctx, claims.SessionID, user.ID); active {
		t.Fatal("logout must revoke the session immediately")
	}
}

func TestPasswordResetFlow(t *testing.T) {
	svc, mail, _, repo := newTestService(t)
	ctx := context.Background()
	email := testutil.Email(t, repo.dbPool, "reset")
	first, user, err := svc.Register(ctx, email, "Password123", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.ForgotPassword(ctx, "unknown_"+email); err != nil || len(mail.Sent) != 0 {
		t.Fatalf("unknown email must succeed silently without mail (err=%v, sent=%d)", err, len(mail.Sent))
	}
	if err := svc.ForgotPassword(ctx, email); err != nil {
		t.Fatalf("forgot: %v", err)
	}
	if len(mail.Sent) != 1 || mail.Sent[0].To != email {
		t.Fatalf("expected one reset email, got %+v", mail.Sent)
	}
	code := regexp.MustCompile(`\b\d{6}\b`).FindString(mail.Sent[0].Body)
	if code == "" {
		t.Fatal("reset email must contain a 6-digit code")
	}
	wrong := "000000"
	if wrong == code {
		wrong = "111111"
	}
	if err := svc.ResetPassword(ctx, email, wrong, "NewPassword1"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("expected invalid code, got %v", err)
	}
	if err := svc.ResetPassword(ctx, email, code, "NewPassword1"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := svc.ResetPassword(ctx, email, code, "OtherPassword1"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("a code must be single use, got %v", err)
	}
	if _, _, err := svc.Refresh(ctx, first.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("reset must revoke existing sessions, got %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "NewPassword1", "", ""); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	_ = user
}

func TestResetCodeLocksAfterTooManyAttempts(t *testing.T) {
	svc, mail, _, repo := newTestService(t)
	ctx := context.Background()
	email := testutil.Email(t, repo.dbPool, "bruteforce")
	if _, _, err := svc.Register(ctx, email, "Password123", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.ForgotPassword(ctx, email); err != nil {
		t.Fatal(err)
	}
	code := regexp.MustCompile(`\b\d{6}\b`).FindString(mail.Sent[0].Body)
	wrong := "000000"
	if wrong == code {
		wrong = "111111"
	}
	for i := 0; i < maxResetCodeAttempts; i++ {
		_ = svc.ResetPassword(ctx, email, wrong, "NewPassword1")
	}
	if err := svc.ResetPassword(ctx, email, code, "NewPassword1"); !errors.Is(err, ErrInvalidResetCode) {
		t.Fatalf("correct code must be refused after too many attempts, got %v", err)
	}
}

func TestDeleteAccountRequiresPasswordAndCleansUp(t *testing.T) {
	svc, _, cleaner, repo := newTestService(t)
	ctx := context.Background()
	email := testutil.Email(t, repo.dbPool, "delete")
	_, user, err := svc.Register(ctx, email, "Password123", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteAccount(ctx, user.ID, "WrongPassword1"); err == nil {
		t.Fatal("wrong password must be refused")
	}
	if err := svc.DeleteAccount(ctx, user.ID, "Password123"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(cleaner.finalized) != 1 || cleaner.finalized[0] != user.ID {
		t.Fatalf("cleaners must run after deletion: %+v", cleaner)
	}
	var remaining int
	_ = repo.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM profiles WHERE user_id = $1`, user.ID).Scan(&remaining)
	if remaining != 0 {
		t.Fatal("profile must be deleted with the account")
	}
	if _, err := svc.Me(ctx, user.ID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected user not found, got %v", err)
	}
}
