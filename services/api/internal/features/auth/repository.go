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

type StoredUserWithPassword struct {
	User         User
	PasswordHash string
}

type StoredReset struct {
	ID       string
	CodeHash string
	Attempts int
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (string, error)
	RotateSession(ctx context.Context, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (sessionID, userID string, err error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetActivePasswordReset(ctx context.Context, userID string) (StoredReset, error)
	IncrementResetAttempts(ctx context.Context, resetID string) error
	CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var user User
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrRepositoryEmailExists
		}
		return User{}, err
	}
	return user, nil
}

func (r *PGRepository) getUserAuth(ctx context.Context, where string, arg string) (StoredUserWithPassword, error) {
	var stored StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE `+where+` = $1
	`, arg).Scan(&stored.User.ID, &stored.User.Email, &stored.PasswordHash, &stored.User.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return stored, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, "email", email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, "id", userID)
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (string, error) {
	var sessionID string
	err := r.dbPool.QueryRow(ctx, `
		WITH touched AS (
			UPDATE users SET last_active_at = NOW() WHERE id = $1 RETURNING id
		)
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		SELECT id, $2, $3, $4, $5, NOW() FROM touched
		RETURNING id
	`, userID, tokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(&sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepositoryNotFound
		}
		return "", err
	}
	return sessionID, nil
}

// RotateSession atomically swaps the refresh token of an active session, so
// a refresh token can only ever be used once.
func (r *PGRepository) RotateSession(ctx context.Context, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (string, string, error) {
	var sessionID, userID string
	err := r.dbPool.QueryRow(ctx, `
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
	`, currentTokenHash, nextTokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(&sessionID, &userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrRepositoryNotFound
		}
		return "", "", err
	}
	_, err = r.dbPool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)
	return sessionID, userID, err
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW(), last_used_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func (r *PGRepository) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&active)
	return active, err
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE password_resets SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
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

func (r *PGRepository) GetActivePasswordReset(ctx context.Context, userID string) (StoredReset, error) {
	var reset StoredReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, code_hash, attempts
		FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&reset.ID, &reset.CodeHash, &reset.Attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredReset{}, ErrRepositoryNotFound
		}
		return StoredReset{}, err
	}
	return reset, nil
}

func (r *PGRepository) IncrementResetAttempts(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

// CompletePasswordReset updates the password, consumes the code and revokes
// every session of the user in one transaction.
func (r *PGRepository) CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE password_resets SET used_at = NOW()
		WHERE id = $1 AND user_id = $2 AND used_at IS NULL
	`, resetID, userID)
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

// DeleteUser removes the account; every personal table cascades.
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
	if len(value) > 512 {
		value = value[:512]
	}
	return value
}

func nullableIP(value string) any {
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.String()
}
