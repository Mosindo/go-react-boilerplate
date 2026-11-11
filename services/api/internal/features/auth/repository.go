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
	ErrRepositoryNotFound    = errors.New("auth record not found")
)

type StoredUser struct {
	User
	PasswordHash string
}

type StoredSession struct {
	ID     string
	UserID string
}

type StoredReset struct {
	ID       string
	CodeHash string
	Attempts int
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (StoredUser, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ip string, expiresAt time.Time) (StoredSession, error)
	RotateSession(ctx context.Context, currentHash, nextHash, userAgent, ip string, expiresAt time.Time) (StoredSession, error)
	RevokeSession(ctx context.Context, tokenHash string) error
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetActivePasswordReset(ctx context.Context, userID string) (StoredReset, error)
	IncrementResetAttempts(ctx context.Context, resetID string) error
	CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, created_at
	`, email, passwordHash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrRepositoryEmailExists
		}
		return User{}, err
	}
	return u, nil
}

func (r *PGRepository) getUser(ctx context.Context, where string, arg string) (StoredUser, error) {
	var u StoredUser
	err := r.db.QueryRow(ctx, `SELECT id, email, created_at, password_hash FROM users WHERE `+where+` = $1`, arg).
		Scan(&u.ID, &u.Email, &u.CreatedAt, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUser{}, ErrRepositoryNotFound
	}
	return u, err
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (StoredUser, error) {
	return r.getUser(ctx, "email", email)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	return r.getUser(ctx, "id", userID)
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ip string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ip), expiresAt).Scan(&s.ID, &s.UserID)
	return s, err
}

// RotateSession atomically swaps the refresh token hash of an active session.
func (r *PGRepository) RotateSession(ctx context.Context, currentHash, nextHash, userAgent, ip string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		RETURNING id, user_id
	`, currentHash, nextHash, nullableString(truncate(userAgent, 255)), nullableIP(ip), expiresAt).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func (r *PGRepository) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Only the latest code is valid.
		if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
		`, userID, codeHash, expiresAt)
		return err
	})
}

func (r *PGRepository) GetActivePasswordReset(ctx context.Context, userID string) (StoredReset, error) {
	var s StoredReset
	err := r.db.QueryRow(ctx, `
		SELECT id, code_hash, attempts FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&s.ID, &s.CodeHash, &s.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredReset{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) IncrementResetAttempts(ctx context.Context, resetID string) error {
	_, err := r.db.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

// CompletePasswordReset sets the new password, burns the code and revokes
// every session of the user in one transaction.
func (r *PGRepository) CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
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
		_, err = tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
		return err
	})
}

// DeleteUser removes the account; foreign keys cascade to profile, photos
// metadata, swipes, matches, messages, blocks, notifications and sessions.
// Reports keep their evidence with nulled user references.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nullableIP(v string) any {
	ip := net.ParseIP(v)
	if ip == nil {
		return nil
	}
	return ip.String()
}

func truncate(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}
