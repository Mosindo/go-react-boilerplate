package auth

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

type captureMailer struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (m *captureMailer) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func newTestService(t *testing.T) (*Service, *PGRepository, *captureMailer) {
	t.Helper()
	pool := testutil.NewDB(t)
	repo := NewPGRepository(pool)
	mail := &captureMailer{}
	svc := NewService(repo, testSecret, mail, bcrypt.MinCost)
	svc.SetSynchronousMail(true)
	return svc, repo, mail
}

func appErr(t *testing.T, err error, status int, code string) {
	t.Helper()
	var ae *httpx.Error
	if !errors.As(err, &ae) || ae.Status != status || ae.Code != code {
		t.Fatalf("want %d/%s, got %v", status, code, err)
	}
}

func TestServiceRegisterLoginRefreshLogout(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()

	tokens, user, err := svc.Register(ctx, "  Carol@Test.Invalid ", "password-123", "agent", "203.0.113.5")
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "carol@test.invalid" || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("unexpected register result: %+v %+v", tokens, user)
	}
	_, _, err = svc.Register(ctx, "carol@test.invalid", "password-123", "", "")
	appErr(t, err, 409, "conflict")

	_, _, err = svc.Login(ctx, "carol@test.invalid", "nope-nope-nope", "", "")
	appErr(t, err, 401, "unauthorized")
	_, _, err = svc.Login(ctx, "ghost@test.invalid", "password-123", "", "")
	appErr(t, err, 401, "unauthorized")

	t2, u2, err := svc.Login(ctx, "CAROL@test.invalid", "password-123", "", "not-an-ip")
	if err != nil || u2.ID != user.ID {
		t.Fatalf("login: %v", err)
	}
	t3, _, err := svc.Refresh(ctx, t2.RefreshToken, "", "")
	if err != nil || t3.RefreshToken == t2.RefreshToken {
		t.Fatalf("refresh must rotate: %v", err)
	}
	_, _, err = svc.Refresh(ctx, t2.RefreshToken, "", "")
	appErr(t, err, 401, "unauthorized")
	if err := svc.Logout(ctx, t3.RefreshToken); err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.Refresh(ctx, t3.RefreshToken, "", "")
	appErr(t, err, 401, "unauthorized")
	_, _, err = svc.Refresh(ctx, "", "", "")
	appErr(t, err, 401, "unauthorized")
}

func TestLoginRunsBcryptForUnknownUsers(t *testing.T) {
	svc, _, _ := newTestService(t)
	cost, err := bcrypt.Cost(svc.dummyHash)
	if err != nil || cost != bcrypt.MinCost {
		t.Fatalf("dummy hash must use the service's cost: %d %v", cost, err)
	}
	if checkPassword(svc.dummyHash, "anything") {
		t.Fatal("dummy hash must never match")
	}
}

func TestResetAttemptsCannotBeRacedPastTheCap(t *testing.T) {
	svc, repo, mail := newTestService(t)
	ctx := context.Background()
	_, user, err := svc.Register(ctx, "dave@test.invalid", "password-123", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ForgotPassword(ctx, "dave@test.invalid"); err != nil {
		t.Fatal(err)
	}
	if len(mail.msgs) != 1 {
		t.Fatalf("expected one mail, got %d", len(mail.msgs))
	}

	var wrongAccepted int32
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := svc.ResetPassword(ctx, "dave@test.invalid", "ZZZZZZZZ", "new-password-123")
			var ae *httpx.Error
			if err == nil || !errors.As(err, &ae) || ae.Status != 400 {
				atomic.AddInt32(&wrongAccepted, 1)
			}
		}()
	}
	wg.Wait()
	if wrongAccepted != 0 {
		t.Fatalf("%d wrong guesses were not answered with 400", wrongAccepted)
	}
	var attempts int
	if err := repo.pool.QueryRow(ctx, `SELECT attempts FROM password_resets WHERE user_id=$1`, user.ID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != ResetMaxAttempts {
		t.Fatalf("attempt counter must stop at %d, got %d", ResetMaxAttempts, attempts)
	}
}

func TestForgotPasswordReplacesOlderCodes(t *testing.T) {
	svc, repo, _ := newTestService(t)
	ctx := context.Background()
	_, user, _ := svc.Register(ctx, "erin@test.invalid", "password-123", "", "")
	for i := 0; i < 3; i++ {
		if err := svc.ForgotPassword(ctx, "erin@test.invalid"); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := repo.pool.QueryRow(ctx, `SELECT count(*) FROM password_resets WHERE user_id=$1 AND used_at IS NULL`, user.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("only the newest reset may stay active, got %d", n)
	}
	if err := svc.ForgotPassword(ctx, "   "); err != nil {
		t.Fatal("blank email must be silently accepted")
	}
	var expires time.Time
	_ = repo.pool.QueryRow(ctx, `SELECT expires_at FROM password_resets WHERE user_id=$1`, user.ID).Scan(&expires)
	if d := time.Until(expires); d < 29*time.Minute || d > 31*time.Minute {
		t.Fatalf("TTL should be 30 minutes, got %v", d)
	}
}
