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

//go:embed schema/*.sql
var schemaFS embed.FS

const migrationLockID int64 = 727001

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

// RunMigrations applies embedded schema/*.sql files in lexical order, once each.
// Applied versions are recorded in schema_migrations; an advisory lock serialises
// concurrent API instances starting at the same time.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := fs.ReadDir(schemaFS, "schema")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("migration lock: %w", err)
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockID) }()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, file := range files {
		var applied bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, file).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", file, err)
		}
		if applied {
			continue
		}
		sqlBytes, err := schemaFS.ReadFile("schema/" + file)
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
	if err := execScript(ctx, tx, sql); err != nil {
		return fmt.Errorf("exec migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	return tx.Commit(ctx)
}

func execScript(ctx context.Context, tx pgx.Tx, sql string) error {
	// Simple protocol (no args) lets a file hold several statements.
	_, err := tx.Exec(ctx, sql)
	return err
}
