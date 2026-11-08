package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryEmailExists = errors.New("repository: email exists")
	ErrRepositoryNotFound    = errors.New("repository: not found")
	ErrRepositorySessionGone = errors.New("repository: session gone")
)

type Repository interface {
	CreateAccount(ctx context.Context, email, passwordHash, firstName string, birthDate time.Time) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserAuth, error)
	GetUserAuthByID(ctx context.Context, id string) (StoredUserAuth, error)
	GetUserByID(ctx context.Context, id string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ip string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, newHash, userAgent, ip string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, at time.Time) error
	RevokeAllSessions(ctx context.Context, userID string, at time.Time) error
	IsSessionActive(ctx context.Context, sessionID string) (bool, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ConsumeResetToken(ctx context.Context, tokenHash string) (string, error)
	DeleteUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) CreateAccount(ctx context.Context, email, hash, firstName string, birth time.Time) (StoredUser, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var u StoredUser
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, created_at`, email, hash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO profiles (user_id, first_name, birth_date) VALUES ($1, $2, $3)`, u.ID, firstName, birth); err != nil {
		return StoredUser{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1)`, u.ID); err != nil {
		return StoredUser{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoredUser{}, err
	}
	return u, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserAuth, error) {
	return r.getAuth(ctx, `WHERE email = $1`, email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, id string) (StoredUserAuth, error) {
	return r.getAuth(ctx, `WHERE id = $1`, id)
}

func (r *PGRepository) getAuth(ctx context.Context, where string, arg any) (StoredUserAuth, error) {
	var u StoredUserAuth
	err := r.db.QueryRow(ctx, `SELECT id, email, created_at, password_hash FROM users `+where, arg).
		Scan(&u.ID, &u.Email, &u.CreatedAt, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUserAuth{}, ErrRepositoryNotFound
	}
	return u, err
}

func (r *PGRepository) GetUserByID(ctx context.Context, id string) (StoredUser, error) {
	var u StoredUser
	err := r.db.QueryRow(ctx, `SELECT id, email, created_at FROM users WHERE id = $1`, id).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUser{}, ErrRepositoryNotFound
	}
	return u, err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ip string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		WITH touched AS (UPDATE users SET last_active_at = NOW() WHERE id = $1)
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4::inet, $5)
		RETURNING id, user_id`, userID, tokenHash, truncate(userAgent, 255), nullIfEmpty(ip), expiresAt).Scan(&s.ID, &s.UserID)
	return s, err
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`, tokenHash).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, newHash, userAgent, ip string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		UPDATE sessions SET token_hash = $2, user_agent = $3, ip_address = $4::inet, expires_at = $5, last_used_at = $6
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id`, sessionID, newHash, truncate(userAgent, 255), nullIfEmpty(ip), expiresAt, usedAt).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string, at time.Time) error {
	tag, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = $2 WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) RevokeAllSessions(ctx context.Context, userID string, at time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`, userID, at)
	return err
}

func (r *PGRepository) IsSessionActive(ctx context.Context, sessionID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM sessions WHERE id = $1 AND revoked_at IS NULL AND expires_at > NOW())`, sessionID).Scan(&ok)
	if err != nil {
		// A malformed (non-UUID) session id in a forged-but-valid token simply is not active.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return false, nil
		}
	}
	return ok, err
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, hash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, hash)
	return err
}

func (r *PGRepository) CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		WITH invalidate AS (UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL)
		INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt)
	return err
}

func (r *PGRepository) ConsumeResetToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx, `
		UPDATE password_resets SET used_at = NOW()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING user_id`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return userID, err
}

// DeleteUser removes the account; every dependent row is removed by ON DELETE CASCADE.
// It returns the storage keys of the user's photos so the caller can delete the files.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) ([]string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrRepositoryNotFound
	}
	return keys, tx.Commit(ctx)
}
