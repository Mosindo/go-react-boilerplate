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
	ID        string
	Email     string
	CreatedAt time.Time
}

type StoredUserWithPassword struct {
	User         StoredUser
	PasswordHash string
}

type StoredSession struct {
	ID         string
	UserID     string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
	CreatePasswordResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ResetPasswordWithToken(ctx context.Context, tokenHash, passwordHash string) (bool, error)
	GetPasswordHash(ctx context.Context, userID string) (string, error)
	DeleteUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	var user StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = $1 AND disabled_at IS NULL
	`, email).Scan(&user.User.ID, &user.User.Email, &user.PasswordHash, &user.User.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, created_at
		FROM users
		WHERE id = $1 AND disabled_at IS NULL
	`, userID).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUser{}, ErrRepositoryNotFound
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		SELECT u.id, $2, $3, $4, $5, NOW()
		FROM users u
		WHERE u.id = $1
		RETURNING id, user_id, expires_at, created_at, last_used_at
	`, userID, tokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositoryNotFound
		}
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.expires_at, s.created_at, s.last_used_at
		FROM sessions s
		INNER JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND u.disabled_at IS NULL
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
	`, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = $6
		WHERE id = $1
		  AND revoked_at IS NULL
		RETURNING id, user_id, expires_at, created_at, last_used_at
	`, sessionID, nextTokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt, usedAt).Scan(
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = $2,
		    last_used_at = $2
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash, revokedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableIP(value string) any {
	if value == "" {
		return nil
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.String()
}

// IsSessionActive reports whether the session is live and its user enabled.
// It also refreshes users.last_active_at at most every five minutes.
func (r *PGRepository) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		WITH live AS (
			SELECT 1
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.id = $2 AND s.user_id = $1
			  AND s.revoked_at IS NULL AND s.expires_at > NOW()
			  AND u.disabled_at IS NULL
		), touch AS (
			UPDATE users SET last_active_at = NOW()
			WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
			  AND EXISTS (SELECT 1 FROM live)
		)
		SELECT EXISTS (SELECT 1 FROM live)
	`, userID, sessionID).Scan(&active)
	return active, err
}

func (r *PGRepository) CreatePasswordResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

// ResetPasswordWithToken consumes a valid reset token, sets the new password
// and revokes every session of the user, atomically.
func (r *PGRepository) ResetPasswordWithToken(ctx context.Context, tokenHash, passwordHash string) (bool, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (r *PGRepository) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	var hash string
	err := r.dbPool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return hash, err
}

// DeleteUser removes the user; every dependent table cascades.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) error {
	tag, err := r.dbPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}
