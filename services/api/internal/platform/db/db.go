package db

import (
	"context"
	"embed"
	"fmt"
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
