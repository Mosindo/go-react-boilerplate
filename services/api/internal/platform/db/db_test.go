package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// scratchDB creates an empty database for the test and returns a pool on it.
func scratchDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("DATABASE_URL_TEST")
	if base == "" {
		t.Skip("DATABASE_URL_TEST not set")
	}
	ctx := context.Background()
	admin, err := Connect(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("migtest_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		admin.Close()
		t.Fatalf("create database: %v", err)
	}
	u, _ := url.Parse(base)
	u.Path = "/" + name
	pool, err := Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		admin.Close()
	})
	return pool
}

func TestMigrationsApplyOnceAndAreIdempotent(t *testing.T) {
	pool := scratchDB(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := RunMigrations(ctx, pool); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	files, _ := migrationFiles()
	var applied int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied)
	if applied != len(files) {
		t.Fatalf("expected %d recorded migrations, got %d", len(files), applied)
	}
	for _, table := range []string{"users", "profiles", "photos", "preferences", "likes", "matches", "conversations", "messages", "blocks", "reports", "notifications", "interests", "user_interests"} {
		var ok bool
		_ = pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&ok)
		if !ok {
			t.Errorf("table %s missing", table)
		}
	}
}

func TestLegacyDatabaseIsBaselinedNotReplayed(t *testing.T) {
	pool := scratchDB(t)
	ctx := context.Background()
	files, _ := migrationFiles()
	for _, f := range files {
		if f > legacyBaseline {
			continue
		}
		sqlBytes, _ := migrationsFS.ReadFile("migrations/" + f)
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("legacy %s: %v", f, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users (email, password_hash) VALUES ('legacy@test.invalid', 'x')`); err != nil {
		// organization_id is required before the dating migration.
		if !strings.Contains(err.Error(), "organization_id") {
			t.Fatal(err)
		}
		_, _ = pool.Exec(ctx, `INSERT INTO users (email, password_hash, organization_id) SELECT 'legacy@test.invalid', 'x', id FROM organizations LIMIT 1`)
	}
	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatalf("upgrade from legacy schema: %v", err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email = 'legacy@test.invalid'`).Scan(&n)
	if n != 1 {
		t.Fatal("existing accounts must survive the upgrade")
	}
}
