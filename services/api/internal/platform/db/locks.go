package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// PairKey is the order-independent key of a relationship between two users.
func PairKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// LockPair serialises every write touching the relationship of two members
// (swipes, match creation, unmatch, block) inside the current transaction so
// concurrent requests cannot produce duplicate or inconsistent matches.
func LockPair(ctx context.Context, tx pgx.Tx, a, b string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('pair:' || $1::text, 0))`, PairKey(a, b))
	return err
}
