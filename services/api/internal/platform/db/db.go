package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

func ConnectWithRetry(ctx context.Context, databaseURL string, maxWait, retryInterval time.Duration) (*pgxpool.Pool, error) {
	if maxWait <= 0 {
		return Connect(ctx, databaseURL)
	}
	if retryInterval <= 0 {
		retryInterval = time.Second
	}

	deadline := time.Now().Add(maxWait)
	var lastErr error

	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		pool, err := Connect(attemptCtx, databaseURL)
		cancel()
		if err == nil {
			return pool, nil
		}
		lastErr = err

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("connect db after retry window %s: %w", maxWait, lastErr)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect db canceled: %w", ctx.Err())
		case <-time.After(retryInterval):
		}
	}
}

// migrationLockID is the pg advisory lock key that serialises concurrent
// migration runs (several API replicas or parallel test packages).
const migrationLockID int64 = 727270001

// RunMigrations applies every embedded *.sql file, in lexical order, that is not
// yet recorded in schema_migrations. Each file runs in its own transaction
// together with its bookkeeping row. Legacy files (001..013) are idempotent, so
// the first tracked run against an already-migrated database is safe.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := migrationFiles()
	if err != nil {
		return err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("take migration lock: %w", err)
	}
	defer func() {
		// Use a fresh context: ctx may already be cancelled.
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationLockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := map[string]struct{}{}
	rows, err := conn.Query(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan applied migration: %w", err)
		}
		applied[name] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}

	for _, file := range files {
		if _, done := applied[file]; done {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + file)
		if err != nil {
			return fmt.Errorf("read embedded migration %s: %w", file, err)
		}
		if err := applyMigration(ctx, conn, file, string(sqlBytes)); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, conn *pgxpool.Conn, name, sql string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("exec migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

func migrationFiles() ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}
