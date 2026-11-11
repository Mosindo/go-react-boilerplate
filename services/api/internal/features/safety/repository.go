package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func userExists(ctx context.Context, q pgx.Tx, userID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&ok)
	return ok, err
}

// Block records the block and dissolves any match (messages cascade), so the
// two members can no longer see or contact each other.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	var removed string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		ok, err := userExists(ctx, tx, blockedID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUserNotFound
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, blockerID, blockedID); err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `
			DELETE FROM matches
			WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
			RETURNING id
		`, blockerID, blockedID).Scan(&removed)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	})
	return removed, err
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2
	`, blockerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BlockedUser, 0)
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CreateReport stores the report with an evidence snapshot (reported profile
// and the latest messages of the shared conversation, if any) taken before
// the automatic block deletes the conversation.
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		ok, err := userExists(ctx, tx, reportedID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUserNotFound
		}
		err = tx.QueryRow(ctx, `
			WITH profile AS (
				SELECT jsonb_build_object(
					'firstName', p.first_name, 'bio', p.bio, 'jobTitle', p.job_title, 'city', p.city,
					'photoKeys', COALESCE((SELECT jsonb_agg(storage_key ORDER BY position) FROM photos WHERE user_id = p.user_id), '[]'::jsonb)
				) AS j
				FROM profiles p WHERE p.user_id = $2
			),
			conv AS (
				SELECT COALESCE(jsonb_agg(jsonb_build_object('senderId', x.sender_id, 'body', x.body, 'createdAt', x.created_at) ORDER BY x.created_at), '[]'::jsonb) AS j
				FROM (
					SELECT msg.sender_id, msg.body, msg.created_at
					FROM matches m JOIN messages msg ON msg.match_id = m.id
					WHERE m.user_low_id = LEAST($1::uuid, $2::uuid) AND m.user_high_id = GREATEST($1::uuid, $2::uuid)
					ORDER BY msg.created_at DESC LIMIT 50
				) x
			)
			INSERT INTO reports (reporter_id, reported_id, reason, details, evidence)
			VALUES ($1, $2, $3, $4, jsonb_build_object(
				'profile', (SELECT j FROM profile),
				'messages', (SELECT j FROM conv)
			))
			ON CONFLICT (reporter_id, reported_id) WHERE status = 'open'
			DO UPDATE SET details = CASE WHEN EXCLUDED.details = '' THEN reports.details ELSE EXCLUDED.details END,
			              reason = EXCLUDED.reason
			RETURNING id
		`, reporterID, reportedID, reason, details).Scan(&id)
		return err
	})
	return id, err
}
