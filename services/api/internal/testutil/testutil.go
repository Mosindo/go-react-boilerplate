// Package testutil provides helpers for integration tests backed by a real
// PostgreSQL database (DATABASE_URL_TEST). Tests are skipped when it is unset.
package testutil

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

var counter atomic.Int64

// DB connects to the test database and applies migrations.
func DB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("integration test skipped: set DATABASE_URL_TEST to run it")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Email returns a unique test email and registers its cleanup.
func Email(t *testing.T, pool *pgxpool.Pool, prefix string) string {
	t.Helper()
	email := fmt.Sprintf("%s_%d_%d@integration.test", prefix, time.Now().UnixNano(), counter.Add(1))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})
	return email
}

// CaptureMailer records sent emails for assertions.
type CaptureMailer struct {
	Sent []Mail
}

type Mail struct {
	To, Subject, Body string
}

func (m *CaptureMailer) Send(_ context.Context, to, subject, body string) error {
	m.Sent = append(m.Sent, Mail{To: to, Subject: subject, Body: body})
	return nil
}
