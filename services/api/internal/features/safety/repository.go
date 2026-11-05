package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type endedMatch struct{ MatchID, ConversationID string }

type Repository interface {
	// Block records the block and ends the match with the user, if any.
	Block(ctx context.Context, blockerID, blockedID string) (*endedMatch, error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func mapFK(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return ErrUserNotFound
	}
	return err
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (*endedMatch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		return nil, mapFK(err)
	}
	var em endedMatch
	err = tx.QueryRow(ctx, `
		UPDATE matches m SET ended_at = NOW(), ended_by = $1
		FROM conversations c
		WHERE c.match_id = m.id AND m.ended_at IS NULL
		  AND m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
		RETURNING m.id, c.id`, blockerID, blockedID).Scan(&em.MatchID, &em.ConversationID)
	var ended *endedMatch
	if err == nil {
		ended = &em
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return ended, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return mapFK(err)
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT $2`, blockerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var u BlockedUser
		if err := rows.Scan(&u.ID, &u.FirstName, &u.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	// One open report per reporter/target/reason per day keeps the queue clean
	// and makes retries idempotent.
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
			SELECT 1 FROM reports
			WHERE reporter_id = $1 AND reported_id = $2 AND reason = $3
			  AND status = 'open' AND created_at > NOW() - INTERVAL '1 day')`,
		reporterID, reportedID, reason, details)
	return mapFK(err)
}
