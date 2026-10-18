package auth_test

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"testing"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/db/dbtest"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func newService(t *testing.T) *auth.Service {
	t.Helper()
	pool := dbtest.New(t)
	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return auth.NewService(auth.NewPGRepository(pool), []byte("0123456789abcdef0123456789abcdef0123"), &mailer.Memory{}, files)
}

func TestRegisterValidatesEmailAndPassword(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()
	cases := []struct {
		name, email, password string
		want                  error
	}{
		{"missing tld", "bob@localhost", "correct-horse-1", auth.ErrInvalidEmail},
		{"display name form", "Bob <bob@example.io>", "correct-horse-1", auth.ErrInvalidEmail},
		{"short password", "bob@example.io", "short", auth.ErrWeakPassword},
		{"password beyond bcrypt limit", "bob@example.io", string(make([]byte, 73)), auth.ErrWeakPassword},
	}
	for _, tc := range cases {
		if _, _, err := svc.Register(ctx, tc.email, tc.password, "", ""); !errors.Is(err, tc.want) {
			t.Errorf("%s: want %v, got %v", tc.name, tc.want, err)
		}
	}
	if _, user, err := svc.Register(ctx, "  Bob@Example.io ", "correct-horse-1", "", ""); err != nil || user.Email != "bob@example.io" {
		t.Fatalf("valid registration failed: %v %v", user, err)
	}
}

// A refresh token can be spent exactly once even when two requests race.
func TestRefreshTokenCanOnlyBeUsedOnceUnderConcurrency(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()
	tokens, _, err := svc.Register(ctx, "race@example.io", "correct-horse-1", "", "")
	if err != nil {
		t.Fatal(err)
	}

	const attempts = 8
	var wg sync.WaitGroup
	results := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.Refresh(ctx, tokens.RefreshToken, "", "")
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	ok := 0
	for err := range results {
		if err == nil {
			ok++
		} else if !errors.Is(err, auth.ErrInvalidRefreshToken) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("exactly one refresh must succeed, got %d", ok)
	}
}
