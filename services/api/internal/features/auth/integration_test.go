package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// memRepo is a minimal in-memory Repository for service-level tests. HTTP and SQL behaviour is covered by
// the database-backed tests in cmd/api.
type memRepo struct {
	users    map[string]StoredUserWithPassword // by email
	sessions map[string]StoredSession          // by token hash
	resets   map[string]string                 // token hash → user id
	deleted  []string
}

func newMemRepo() *memRepo {
	return &memRepo{users: map[string]StoredUserWithPassword{}, sessions: map[string]StoredSession{}, resets: map[string]string{}}
}

func (m *memRepo) CreateUser(_ context.Context, email, hash string) (StoredUser, error) {
	if _, ok := m.users[email]; ok {
		return StoredUser{}, ErrRepositoryEmailExists
	}
	u := StoredUser{ID: "user-" + email, Email: email, CreatedAt: time.Now()}
	m.users[email] = StoredUserWithPassword{User: u, PasswordHash: hash}
	return u, nil
}
func (m *memRepo) GetUserAuthByEmail(_ context.Context, email string) (StoredUserWithPassword, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return StoredUserWithPassword{}, ErrRepositoryNotFound
}
func (m *memRepo) GetUserAuthByID(_ context.Context, id string) (StoredUserWithPassword, error) {
	for _, u := range m.users {
		if u.User.ID == id {
			return u, nil
		}
	}
	return StoredUserWithPassword{}, ErrRepositoryNotFound
}
func (m *memRepo) GetUserByID(ctx context.Context, id string) (StoredUser, error) {
	u, err := m.GetUserAuthByID(ctx, id)
	return u.User, err
}
func (m *memRepo) CreateSession(_ context.Context, userID, hash, _, _ string, exp time.Time) (StoredSession, error) {
	s := StoredSession{ID: "sess-" + hash[:6], UserID: userID, ExpiresAt: exp}
	m.sessions[hash] = s
	return s, nil
}
func (m *memRepo) GetActiveSessionByTokenHash(_ context.Context, hash string) (StoredSession, error) {
	if s, ok := m.sessions[hash]; ok {
		return s, nil
	}
	return StoredSession{}, ErrRepositorySessionGone
}
func (m *memRepo) RotateSessionToken(_ context.Context, id, current, next, _, _ string, _, _ time.Time) (StoredSession, error) {
	s, ok := m.sessions[current]
	if !ok || s.ID != id {
		return StoredSession{}, ErrRepositorySessionGone
	}
	delete(m.sessions, current)
	m.sessions[next] = s
	return s, nil
}
func (m *memRepo) RevokeSessionByTokenHash(_ context.Context, hash string, _ time.Time) error {
	if _, ok := m.sessions[hash]; !ok {
		return ErrRepositorySessionGone
	}
	delete(m.sessions, hash)
	return nil
}
func (m *memRepo) IsSessionActive(context.Context, string, string) (bool, error) { return true, nil }
func (m *memRepo) CreatePasswordReset(_ context.Context, userID, hash string, _ time.Time) error {
	m.resets[hash] = userID
	return nil
}
func (m *memRepo) ConsumePasswordReset(_ context.Context, hash, newHash string) (string, error) {
	id, ok := m.resets[hash]
	if !ok {
		return "", ErrRepositoryTokenGone
	}
	delete(m.resets, hash)
	for email, u := range m.users {
		if u.User.ID == id {
			u.PasswordHash = newHash
			m.users[email] = u
		}
	}
	return id, nil
}
func (m *memRepo) UpdatePassword(context.Context, string, string, string) error { return nil }
func (m *memRepo) DeleteUser(_ context.Context, id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}

type recordingMailer struct{ to, body []string }

func (r *recordingMailer) Send(_ context.Context, to, _, body string) error {
	r.to, r.body = append(r.to, to), append(r.body, body)
	return nil
}

type fakeEraser struct{ prepared, finished int }

func (f *fakeEraser) Prepare(context.Context, string) (func(), error) {
	f.prepared++
	return func() { f.finished++ }, nil
}

func newTestService(repo Repository, opts ...Option) *Service {
	return NewService(repo, []byte("service-test-secret-0123456789abcdef"), opts...)
}

func TestRegisterNormalisesAndValidates(t *testing.T) {
	svc := newTestService(newMemRepo())
	ctx := context.Background()

	_, user, err := svc.Register(ctx, "  Alice@Example.COM ", "correct-horse", "ua", "1.2.3.4")
	if err != nil || user.Email != "alice@example.com" {
		t.Fatalf("email must be trimmed and lower-cased: %+v %v", user, err)
	}
	if _, _, err := svc.Register(ctx, "alice@example.com", "correct-horse", "", ""); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("duplicate: %v", err)
	}
	for _, email := range []string{"", "plain", "a@b", "a b@c.de", "<a@b.cd>", "Name <a@b.cd>", "a@b.", "@b.cd", "a@b.cd,e@f.gh"} {
		if _, _, err := svc.Register(ctx, email, "correct-horse", "", ""); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("%q must be rejected, got %v", email, err)
		}
	}
	for _, pw := range []string{"", "1234567", string(make([]byte, 73))} {
		if _, _, err := svc.Register(ctx, "bob@example.com", pw, "", ""); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("password of %d bytes must be rejected, got %v", len(pw), err)
		}
	}
}

func TestAccessTokenClaims(t *testing.T) {
	svc := newTestService(newMemRepo())
	tokens, user, err := svc.Register(context.Background(), "claims@example.com", "correct-horse", "", "")
	if err != nil {
		t.Fatal(err)
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(tokens.AccessToken, claims, func(*jwt.Token) (any, error) { return svc.jwtSecret, nil })
	if err != nil || !parsed.Valid || parsed.Method.Alg() != "HS256" {
		t.Fatalf("token must be a valid HS256 JWT: %v", err)
	}
	exp, _ := claims.GetExpirationTime()
	if exp == nil || time.Until(exp.Time) > accessTokenTTL || time.Until(exp.Time) < accessTokenTTL-time.Minute {
		t.Fatalf("access token must expire in about %s, got %v", accessTokenTTL, exp)
	}
	if claims["uid"] != user.ID || claims["sid"] == "" {
		t.Fatalf("missing identity claims: %v", claims)
	}
}

func TestLoginDoesNotRevealWhetherEmailExists(t *testing.T) {
	svc := newTestService(newMemRepo())
	ctx := context.Background()
	_, _, _ = svc.Register(ctx, "known@example.com", "correct-horse", "", "")

	_, _, errWrong := svc.Login(ctx, "known@example.com", "bad-password", "", "")
	_, _, errUnknown := svc.Login(ctx, "unknown@example.com", "bad-password", "", "")
	if !errors.Is(errWrong, ErrInvalidCredentials) || !errors.Is(errUnknown, ErrInvalidCredentials) {
		t.Fatalf("both failures must look identical: %v / %v", errWrong, errUnknown)
	}
}

func TestPasswordRecoveryService(t *testing.T) {
	repo := newMemRepo()
	mail := &recordingMailer{}
	svc := newTestService(repo, WithMailer(mail, "app://reset?token={token}"), WithSyncMail())
	ctx := context.Background()
	_, _, _ = svc.Register(ctx, "forgot@example.com", "correct-horse", "", "")

	if err := svc.ForgotPassword(ctx, "nobody@example.com"); err != nil || len(mail.to) != 0 {
		t.Fatalf("unknown email: no error and no mail expected: %v %v", err, mail.to)
	}
	if err := svc.ForgotPassword(ctx, " FORGOT@example.com "); err != nil || len(mail.to) != 1 || mail.to[0] != "forgot@example.com" {
		t.Fatalf("expected one mail: %v %v", err, mail.to)
	}
	if len(repo.resets) != 1 {
		t.Fatal("reset token must be stored")
	}
	for hash := range repo.resets {
		if len(hash) != 64 {
			t.Fatal("only a SHA-256 hash of the token may be stored")
		}
	}
	if err := svc.ResetPassword(ctx, "wrong", "new-password-1"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("unknown token: %v", err)
	}
}

func TestDeleteAccountRunsEraser(t *testing.T) {
	repo := newMemRepo()
	eraser := &fakeEraser{}
	svc := newTestService(repo, WithAccountEraser(eraser))
	ctx := context.Background()
	_, user, _ := svc.Register(ctx, "bye@example.com", "correct-horse", "", "")

	if err := svc.DeleteAccount(ctx, user.ID, "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	if eraser.prepared != 0 || len(repo.deleted) != 0 {
		t.Fatal("nothing may be touched when the password is wrong")
	}
	if err := svc.DeleteAccount(ctx, user.ID, "correct-horse"); err != nil {
		t.Fatal(err)
	}
	if eraser.prepared != 1 || eraser.finished != 1 || len(repo.deleted) != 1 {
		t.Fatalf("eraser and repo must both run: %+v %v", eraser, repo.deleted)
	}
}

func TestRefreshTokenIsSingleUse(t *testing.T) {
	svc := newTestService(newMemRepo())
	ctx := context.Background()
	tokens, _, _ := svc.Register(ctx, "refresh@example.com", "correct-horse", "", "")

	next, _, err := svc.Refresh(ctx, tokens.RefreshToken, "", "")
	if err != nil || next.RefreshToken == tokens.RefreshToken {
		t.Fatalf("rotation failed: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, tokens.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("old refresh token must be refused: %v", err)
	}
}
