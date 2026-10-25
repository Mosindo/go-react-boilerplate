// Package testutil provides an isolated PostgreSQL schema per test package.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool connects to TEST_DATABASE_URL (or DATABASE_URL), creates a throwaway schema,
// runs the migrations in it and drops it at cleanup. The test is skipped when no database is configured.
func NewPool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL is required for integration tests")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}

	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	schema := "t_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatalf("create schema: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		admin.Close()
		t.Fatalf("parse config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		admin.Close()
		t.Fatalf("connect pool: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		pool.Close()
		admin.Close()
		t.Fatalf("migrations: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	return pool
}
