package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryUserNotFound = errors.New("user not found")

type Repository interface {
	// Block records the block and removes any match (and its messages) between the two users.
	// It returns the id of the removed match, if any.
	Block(ctx context.Context, blockerID, blockedID string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit int) ([]StoredBlock, error)
	// Report stores a report with a snapshot of the last messages exchanged, if matched.
	// It is idempotent per (reporter, reported, reason) within 24 hours.
	Report(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func userExists(ctx context.Context, tx pgx.Tx, id string) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1::uuid)`, id).Scan(&ok)
	return ok, err
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ok, err := userExists(ctx, tx, blockedID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrRepositoryUserNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return "", err
	}
	var matchID *string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
		RETURNING id
	`, blockerID, blockedID).Scan(&matchID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if matchID != nil {
		return *matchID, nil
	}
	return "", nil
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1::uuid AND blocked_id = $2::uuid`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit int) ([]StoredBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT blocked_id, created_at FROM blocks WHERE blocker_id = $1 ORDER BY created_at DESC LIMIT $2
	`, blockerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredBlock{}
	for rows.Next() {
		var b StoredBlock
		if err := rows.Scan(&b.UserID, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ok, err := userExists(ctx, tx, reportedID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrRepositoryUserNotFound
	}
	var duplicate bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM reports
		  WHERE reporter_id = $1::uuid AND reported_id = $2::uuid AND reason = $3
		    AND created_at > NOW() - INTERVAL '24 hours')
	`, reporterID, reportedID, reason).Scan(&duplicate); err != nil {
		return err
	}
	if duplicate {
		return tx.Commit(ctx)
	}
	// Evidence: the last 20 messages of the shared conversation, kept even if the
	// conversation is later deleted by a block.
	if _, err := tx.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details, context)
		VALUES ($1::uuid, $2::uuid, $3, $4, COALESCE((
			SELECT jsonb_agg(jsonb_build_object('senderId', t.sender_id, 'body', t.body, 'createdAt', t.created_at)
			                 ORDER BY t.created_at)
			FROM (
				SELECT cm.sender_id, cm.body, cm.created_at
				FROM chat_messages cm
				JOIN matches m ON m.id = cm.match_id
				WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
				ORDER BY cm.created_at DESC LIMIT 20
			) t
		), '[]'::jsonb))
	`, reporterID, reportedID, reason, details); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
