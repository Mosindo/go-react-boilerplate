package safety

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound       = errors.New("safety: user not found")
	ErrSelf           = errors.New("safety: cannot target yourself")
	ErrTooManyReports = errors.New("safety: too many reports")
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (*DissolvedMatch, error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]BlockItem, error)
	CreateReport(ctx context.Context, reporterID, reportedID string, reason Reason, details string) (string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Block records the block and dissolves any active match in one transaction.
// Blocked users disappear from each other's discovery, matches, chats and photos.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (*DissolvedMatch, error) {
	if blockerID == blockedID {
		return nil, ErrSelf
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		if isFK(err) || isInvalidUUID(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var dm DissolvedMatch
	err = tx.QueryRow(ctx, `
		UPDATE matches m SET unmatched_at = NOW()
		WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid) AND m.unmatched_at IS NULL
		RETURNING m.id::text, m.user_a::text, m.user_b::text, COALESCE((SELECT c.id::text FROM conversations c WHERE c.match_id = m.id), '')
	`, blockerID, blockedID).Scan(&dm.MatchID, &dm.UserIDs[0], &dm.UserIDs[1], &dm.ConversationID)
	var result *DissolvedMatch
	switch {
	case err == nil:
		result = &dm
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	if isInvalidUUID(err) {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]BlockItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+profiles.SummarySelect+`, b.created_at
		FROM blocks b JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 200
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockItem{}
	for rows.Next() {
		var it BlockItem
		if err := rows.Scan(&it.User.ID, &it.User.FirstName, &it.User.Age, &it.User.MainPhotoURL, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID string, reason Reason, details string) (string, error) {
	if reporterID == reportedID {
		return "", ErrSelf
	}
	var recent int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND created_at > NOW() - INTERVAL '24 hours'
	`, reporterID).Scan(&recent); err != nil {
		return "", err
	}
	if recent >= maxReportsPerDay {
		return "", ErrTooManyReports
	}
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4) RETURNING id::text
	`, reporterID, reportedID, string(reason), details).Scan(&id)
	if err != nil {
		if isFK(err) || isInvalidUUID(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	return id, nil
}

func isFK(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
