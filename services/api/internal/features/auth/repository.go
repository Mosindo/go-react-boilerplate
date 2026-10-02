package auth

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryEmailExists = errors.New("auth email already exists")
	ErrRepositoryNotFound    = errors.New("auth user not found")
	ErrRepositorySessionGone = errors.New("auth session not found")
)

type StoredUser struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type StoredSession struct {
	ID     string
	UserID string
}

type StoredReset struct {
	ID       string
	UserID   string
	CodeHash string
	Attempts int
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserByEmail(ctx context.Context, email string) (StoredUser, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) error
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RevokeUserSessions(ctx context.Context, userID, exceptSessionID string) error
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetActivePasswordReset(ctx context.Context, userID string) (StoredReset, error)
	RecordResetAttempt(ctx context.Context, resetID string) error
	CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	// DeleteUser removes the user (all related rows cascade) and returns the storage keys of their photos.
	DeleteUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	var u StoredUser
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at`, email, passwordHash).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	return u, nil
}

func (r *PGRepository) getUser(ctx context.Context, where string, arg string) (StoredUser, error) {
	var u StoredUser
	err := r.db.QueryRow(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE `+where, arg).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUser{}, ErrRepositoryNotFound
	}
	return u, err
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (StoredUser, error) {
	return r.getUser(ctx, `email = $1`, email)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	return r.getUser(ctx, `id = $1`, userID)
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING id, user_id`,
		userID, tokenHash, nullableString(truncate(userAgent, 300)), nullableIP(ipAddress), expiresAt).
		Scan(&s.ID, &s.UserID)
	return s, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`, tokenHash).
		Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

// RotateSessionToken swaps the refresh token only if the presented one is still current,
// so two concurrent refreshes with the same token cannot both succeed.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE sessions SET token_hash = $2, user_agent = COALESCE($3, user_agent),
		       ip_address = COALESCE($4, ip_address), expires_at = $5, last_used_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL`,
		sessionID, nextTokenHash, nullableString(truncate(userAgent, 300)), nullableIP(ipAddress), expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL
		RETURNING id, user_id`, tokenHash).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

func (r *PGRepository) RevokeUserSessions(ctx context.Context, userID, exceptSessionID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL AND ($2 = '' OR id::text <> $2)`, userID, exceptSessionID)
	return err
}

// SessionActive implements middleware.SessionChecker and refreshes last_active_at at most once a minute.
func (r *PGRepository) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM sessions
		  WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW())`, sessionID, userID).Scan(&ok)
	if err != nil || !ok {
		return false, err
	}
	_, _ = r.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'`, userID)
	return true, nil
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetActivePasswordReset(ctx context.Context, userID string) (StoredReset, error) {
	var s StoredReset
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, code_hash, attempts FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1`, userID).Scan(&s.ID, &s.UserID, &s.CodeHash, &s.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredReset{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) RecordResetAttempt(ctx context.Context, resetID string) error {
	_, err := r.db.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

func (r *PGRepository) CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, resetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	return err
}

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
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return nil, err
	}
	return keys, tx.Commit(ctx)
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func truncate(v string, n int) string {
	if len(v) > n {
		return v[:n]
	}
	return v
}

func nullableIP(v string) any {
	if ip := net.ParseIP(v); ip != nil {
		return ip.String()
	}
	return nil
}
