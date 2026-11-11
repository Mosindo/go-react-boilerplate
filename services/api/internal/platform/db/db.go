package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrationLockID is an arbitrary constant used with pg_advisory_lock so that
// concurrent API instances (or parallel test packages) never apply migrations
// at the same time.
const migrationLockID int64 = 727_100_042

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

// MigrationNames returns the embedded migration file names in apply order.
func MigrationNames() ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// RunMigrations applies every embedded migration that has not been recorded in
// schema_migrations yet. Each migration runs in its own transaction and the
// whole run is serialized with a Postgres advisory lock.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	names, err := MigrationNames()
	if err != nil {
		return err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := make(map[string]struct{})
	rows, err := conn.Query(ctx, `SELECT name FROM schema_migrations`)
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

	for _, name := range names {
		if _, done := applied[name]; done {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read embedded migration %s: %w", name, err)
		}
		if err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name)
			return err
		}); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}
