package safety

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserMissing = errors.New("user missing")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) UserExists(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&ok)
	return ok, err
}

// Block records the block and removes any match (and therefore the conversation) between the pair.
// It returns the removed conversation ids so clients can be told.
func (r *Repository) Block(ctx context.Context, blockerID, blockedID string) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		WITH removed AS (
			DELETE FROM matches
			WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
			RETURNING id
		)
		SELECT c.id FROM conversations c JOIN removed ON removed.id = c.match_id
	`, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	var conversations []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		conversations = append(conversations, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conversations, tx.Commit(ctx)
}

func (r *Repository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *Repository) Blocked(ctx context.Context, blockerID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT blocked_id FROM blocks WHERE blocker_id = $1 ORDER BY created_at DESC`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CreateReport stores the report together with a snapshot of the latest messages exchanged
// with the reported user, so moderators keep the evidence even if the chat is later deleted.
func (r *Repository) CreateReport(ctx context.Context, reporterID, targetID, reason, details string) (string, error) {
	var context_ []byte
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(jsonb_agg(jsonb_build_object(
		         'senderId', m.sender_id, 'content', m.content, 'createdAt', m.created_at) ORDER BY m.created_at), '[]'::jsonb)
		FROM (
			SELECT m.* FROM messages m
			JOIN conversations c ON c.id = m.conversation_id
			JOIN matches mt ON mt.id = c.match_id
			WHERE mt.user_a = LEAST($1::uuid, $2::uuid) AND mt.user_b = GREATEST($1::uuid, $2::uuid)
			ORDER BY m.created_at DESC LIMIT 20
		) m
	`, reporterID, targetID).Scan(&context_)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if len(context_) == 0 {
		context_ = json.RawMessage("[]")
	}
	var id string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, target_id, reason, details, context) VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, reporterID, targetID, reason, details, context_).Scan(&id)
	return id, err
}
