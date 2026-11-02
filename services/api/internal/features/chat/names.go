package chat

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGNames implements NameLookup.
type PGNames struct{ pool *pgxpool.Pool }

func NewPGNames(pool *pgxpool.Pool) *PGNames { return &PGNames{pool: pool} }

func (n *PGNames) FirstName(ctx context.Context, userID string) (string, error) {
	var name string
	err := n.pool.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, userID).Scan(&name)
	return name, err
}
