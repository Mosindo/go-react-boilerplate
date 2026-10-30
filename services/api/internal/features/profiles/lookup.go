package profiles

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is satisfied by *pgxpool.Pool and pgx.Tx, so lookups can run inside
// a transaction or straight on the pool.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// LoadPhotoIDs returns each user's photo ids ordered by position, in a single
// query regardless of how many users are requested.
func LoadPhotoIDs(ctx context.Context, q Querier, userIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT user_id::text, id::text FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, photoID string
		if err := rows.Scan(&userID, &photoID); err != nil {
			return nil, err
		}
		out[userID] = append(out[userID], photoID)
	}
	return out, rows.Err()
}

// LoadInterests returns each user's interests ordered by label.
func LoadInterests(ctx context.Context, q Querier, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT ui.user_id::text, i.id, i.slug, i.label
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY ui.user_id, i.label`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var it Interest
		if err := rows.Scan(&userID, &it.ID, &it.Slug, &it.Label); err != nil {
			return nil, err
		}
		out[userID] = append(out[userID], it)
	}
	return out, rows.Err()
}
