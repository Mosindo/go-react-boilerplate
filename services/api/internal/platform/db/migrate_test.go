package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tempDB creates an empty database and returns a pool on it.
func tempDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	admin := *u
	admin.Path = "/postgres"
	adminPool, err := pgxpool.New(ctx, admin.String())
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	name := "t_mig_" + hex.EncodeToString(buf)
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		adminPool.Close()
		t.Skipf("cannot create a scratch database: %v", err)
	}
	child := *u
	child.Path = "/" + name
	pool, err := pgxpool.New(ctx, child.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, _ = adminPool.Exec(c, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		adminPool.Close()
	})
	return pool
}

func appliedCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRunMigrationsTracksAndIsIdempotent(t *testing.T) {
	pool := tempDB(t)
	ctx := context.Background()
	files, err := migrationFiles()
	if err != nil || len(files) < 15 {
		t.Fatalf("embedded migrations: %v %v", files, err)
	}
	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if got := appliedCount(t, pool); got != len(files) {
		t.Fatalf("schema_migrations has %d rows, want %d", got, len(files))
	}
	var before time.Time
	if err := pool.QueryRow(ctx, `SELECT max(applied_at) FROM schema_migrations`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatalf("second run must be a no-op: %v", err)
	}
	var after time.Time
	_ = pool.QueryRow(ctx, `SELECT max(applied_at) FROM schema_migrations`).Scan(&after)
	if !after.Equal(before) || appliedCount(t, pool) != len(files) {
		t.Fatal("second run re-applied migrations")
	}

	// the dating schema exists and users carry no organization
	for _, table := range []string{"profiles", "preferences", "interests", "user_interests", "photos", "swipes", "matches", "match_conversations", "match_messages", "blocks", "reports", "notifications", "password_resets", "sessions"} {
		var ok bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&ok); err != nil || !ok {
			t.Errorf("table %s missing (%v)", table, err)
		}
	}
	var hasOrg bool
	_ = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='organization_id')`).Scan(&hasOrg)
	if hasOrg {
		t.Error("users.organization_id must be gone")
	}
	var interests int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM interests`).Scan(&interests)
	if interests < 25 {
		t.Errorf("interest catalogue not seeded: %d", interests)
	}
}

// A database migrated by the old, untracked runner (files 001..013 only) must
// upgrade in place, keeping its users.
func TestRunMigrationsUpgradesLegacyDatabase(t *testing.T) {
	pool := tempDB(t)
	ctx := context.Background()
	files, _ := migrationFiles()
	for _, f := range files {
		if f >= "014" {
			break
		}
		b, err := migrationsFS.ReadFile("migrations/" + f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(b)); err != nil {
			t.Fatalf("legacy %s: %v", f, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users (email, password_hash, organization_id) SELECT 'old@test.invalid', 'x', id FROM organizations WHERE slug='default'`); err != nil {
		t.Fatal(err)
	}

	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email='old@test.invalid'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("legacy user lost: %d %v", n, err)
	}
	// a registration without any organization works
	if _, err := pool.Exec(ctx, `INSERT INTO users (email, password_hash) VALUES ('new@test.invalid', 'x')`); err != nil {
		t.Fatalf("insert without organization: %v", err)
	}
	if got := appliedCount(t, pool); got != len(files) {
		t.Fatalf("tracked %d of %d", got, len(files))
	}
}

func TestRunMigrationsConcurrentRunnersSerialise(t *testing.T) {
	pool := tempDB(t)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- RunMigrations(context.Background(), pool)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent migration failed: %v", err)
		}
	}
	files, _ := migrationFiles()
	if got := appliedCount(t, pool); got != len(files) {
		t.Fatalf("tracked %d of %d", got, len(files))
	}
}

func TestMigrationFilesAreOrderedAndNumbered(t *testing.T) {
	files, err := migrationFiles()
	if err != nil {
		t.Fatal(err)
	}
	prev := ""
	for _, f := range files {
		if len(f) < 4 || f[3] != '_' || strings.Trim(f[:3], "0123456789") != "" {
			t.Errorf("migration %q must be named NNN_description.sql", f)
		}
		if f[:3] == prev {
			t.Errorf("duplicate migration number %s", prev)
		}
		prev = f[:3]
	}
}
