package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoUserNotFound = errors.New("user not found")
	ErrRepoBadMatch     = errors.New("conversation does not involve both users")
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	List(ctx context.Context, blockerID string) ([]Block, error)
	Report(ctx context.Context, reporterID, reportedID, matchID, reason, details string) (string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID)
	if isForeignKey(err) {
		return ErrRepoUserNotFound
	}
	return err
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) List(ctx context.Context, blockerID string) ([]Block, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 500`, blockerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Block, error) {
		var b Block
		err := row.Scan(&b.UserID, &b.FirstName, &b.CreatedAt)
		return b, err
	})
}

func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, matchID, reason, details string) (string, error) {
	if matchID != "" {
		var ok bool
		if err := r.db.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM matches WHERE id = $1
			  AND user_a = LEAST($2::uuid, $3::uuid) AND user_b = GREATEST($2::uuid, $3::uuid))`,
			matchID, reporterID, reportedID).Scan(&ok); err != nil {
			return "", err
		}
		if !ok {
			return "", ErrRepoBadMatch
		}
	}
	var id string
	var match any
	if matchID != "" {
		match = matchID
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, match_id, reason, details)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`, reporterID, reportedID, match, reason, details).Scan(&id)
	if isForeignKey(err) {
		return "", ErrRepoUserNotFound
	}
	return id, err
}

func isForeignKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
