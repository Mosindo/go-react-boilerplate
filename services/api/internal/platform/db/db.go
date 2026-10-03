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

const migrationLockID int64 = 704211001

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

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	// A session-level advisory lock serializes concurrent API instances (and parallel tests).
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID)
	}()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, file := range files {
		var applied bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)", file).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", file, err)
		}
		if applied {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + file)
		if err != nil {
			return fmt.Errorf("read embedded migration %s: %w", file, err)
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", file, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("exec migration %s: %w", file, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", file); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", file, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", file, err)
		}
	}

	return nil
}
