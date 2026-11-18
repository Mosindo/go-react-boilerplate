package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("user not found")

type Repository interface {
	GetPasswordHash(ctx context.Context, userID string) (string, error)
	DeleteUser(ctx context.Context, userID string) (DeletedAccount, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	var hash string
	err := r.dbPool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return hash, err
}

// DeleteUser erases the account and every piece of personal data in one
// transaction. Foreign keys cascade profiles, preferences, photos, interests,
// swipes, matches, blocks, sessions, messages and notifications; reports keep
// their moderation history with the user reference set to NULL.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) (DeletedAccount, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return DeletedAccount{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var deleted DeletedAccount
	if err := collect(ctx, tx, &deleted.PhotoStorageKeys, `SELECT storage_key FROM photos WHERE user_id = $1`, userID); err != nil {
		return DeletedAccount{}, err
	}
	if err := collect(ctx, tx, &deleted.FormerMatchIDs, `
		SELECT CASE WHEN user_low_id = $1 THEN user_high_id ELSE user_low_id END::text
		FROM matches
		WHERE user_low_id = $1 OR user_high_id = $1
	`, userID); err != nil {
		return DeletedAccount{}, err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM conversations
		WHERE id IN (SELECT conversation_id FROM conversation_participants WHERE user_id = $1)
	`, userID); err != nil {
		return DeletedAccount{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE data->>'userId' = $1::text`, userID); err != nil {
		return DeletedAccount{}, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return DeletedAccount{}, err
	}
	if tag.RowsAffected() == 0 {
		return DeletedAccount{}, ErrRepositoryNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return DeletedAccount{}, err
	}
	return deleted, nil
}

func collect(ctx context.Context, tx pgx.Tx, dst *[]string, query string, args ...any) error {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		*dst = append(*dst, v)
	}
	return rows.Err()
}
