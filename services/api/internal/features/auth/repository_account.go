package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// maxResetAttempts bounds guesses of the 6-digit recovery code.
const maxResetAttempts = 5

type StoredPasswordReset struct {
	ID       string
	CodeHash string
}

func (r *PGRepository) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&active)
	return active, err
}

func (r *PGRepository) RevokeAllSessions(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}

func (r *PGRepository) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	var hash string
	err := r.dbPool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return hash, err
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, userID, passwordHash)
	return err
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL
	`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetActivePasswordReset(ctx context.Context, userID string) (StoredPasswordReset, error) {
	var reset StoredPasswordReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, code_hash FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW() AND attempts < $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, maxResetAttempts).Scan(&reset.ID, &reset.CodeHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredPasswordReset{}, ErrRepositoryNotFound
	}
	return reset, err
}

func (r *PGRepository) RecordPasswordResetAttempt(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

// CompletePasswordReset consumes the code exactly once, sets the new password
// and revokes every session in a single transaction.
func (r *PGRepository) CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL
	`, resetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL
	`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteUser removes the account; foreign keys cascade to every owned row
// (profile, photos, likes, matches, messages, notifications, sessions, ...).
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}
