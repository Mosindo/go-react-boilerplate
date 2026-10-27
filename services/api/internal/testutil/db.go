// Package testutil holds helpers shared by integration tests.
package testutil

import (
	"context"
	"os"
	"testing"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB connects to DATABASE_URL_TEST (or DATABASE_URL), applies migrations and
// returns the pool. Tests are skipped when no database is configured so that
// `go test ./...` stays usable without Postgres; CI always provides one.
func DB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("skip integration test: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("skip integration test: postgres unavailable (%v)", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
