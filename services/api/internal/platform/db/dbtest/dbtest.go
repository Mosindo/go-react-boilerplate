// Package dbtest gives each test its own freshly migrated PostgreSQL database,
// so packages can run in parallel without sharing state.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func baseURL(t testing.TB) string {
	t.Helper()
	raw := os.Getenv("DATABASE_URL_TEST")
	if raw == "" {
		raw = os.Getenv("DATABASE_URL")
	}
	if raw == "" {
		t.Skip("integration tests need DATABASE_URL_TEST or DATABASE_URL")
	}
	return raw
}

// New creates an isolated database, applies every migration and returns a pool.
// The database is dropped when the test ends.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	base := baseURL(t)
	admin, err := db.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect admin database: %v", err)
	}

	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	name := "test_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		admin.Close()
		t.Fatalf("create test database: %v", err)
	}

	u, err := url.Parse(base)
	if err != nil {
		admin.Close()
		t.Fatalf("parse database url: %v", err)
	}
	u.Path = "/" + name
	pool, err := db.Connect(ctx, u.String())
	if err != nil {
		admin.Close()
		t.Fatalf("connect test database: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		pool.Close()
		admin.Close()
		t.Fatalf("migrate test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
		admin.Close()
	})
	return pool
}
