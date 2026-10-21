// Package testutil provides database helpers for integration tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool connects to TEST_DATABASE_URL, applies migrations and returns the pool.
// Tests are skipped (with an explicit message) when the variable is not set.
func Pool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping database integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("migrate test db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Email returns a unique throwaway address.
func Email(prefix string) string {
	return fmt.Sprintf("%s-%d@test.alma.invalid", prefix, time.Now().UnixNano())
}
