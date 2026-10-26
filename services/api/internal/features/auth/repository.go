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
	ErrRepositoryResetGone   = errors.New("auth password reset not found")
)

type StoredUser struct {
	ID        string
	Email     string
	Status    string
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

type StoredReset struct {
	ID        string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetLatestPasswordReset(ctx context.Context, userID string) (StoredReset, error)
	BumpPasswordResetAttempts(ctx context.Context, resetID string) error
	// CompletePasswordReset sets the new password, consumes the reset and revokes every session.
	CompletePasswordReset(ctx context.Context, userID, resetID, passwordHash string) error
	// UpdatePassword sets a new password and revokes every session except keepSessionID.
	UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error
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
		RETURNING id, email, status, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.Status, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, "email", email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, "id::text", userID)
}

func (r *PGRepository) getUserAuth(ctx context.Context, column, value string) (StoredUserWithPassword, error) {
	var user StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, status, password_hash, created_at
		FROM users
		WHERE `+column+` = $1
	`, value).Scan(&user.User.ID, &user.User.Email, &user.User.Status, &user.PasswordHash, &user.User.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	stored, err := r.GetUserAuthByID(ctx, userID)
	return stored.User, err
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id, expires_at, created_at, last_used_at
	`, userID, tokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(
		&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &session.LastUsedAt,
	)
	if err != nil {
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id, expires_at, created_at, last_used_at
		FROM sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`, tokenHash).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &session.LastUsedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = $6
		WHERE id = $1
		  AND token_hash = $7
		  AND revoked_at IS NULL
		RETURNING id, user_id, expires_at, created_at, last_used_at
	`, sessionID, nextTokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt, usedAt, currentTokenHash).Scan(
		&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &session.LastUsedAt,
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
		SET revoked_at = $2, last_used_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash, revokedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Only the latest code is valid.
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetLatestPasswordReset(ctx context.Context, userID string) (StoredReset, error) {
	var reset StoredReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, code_hash, attempts, expires_at, created_at
		FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&reset.ID, &reset.CodeHash, &reset.Attempts, &reset.ExpiresAt, &reset.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredReset{}, ErrRepositoryResetGone
		}
		return StoredReset{}, err
	}
	return reset, nil
}

func (r *PGRepository) BumpPasswordResetAttempts(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

func (r *PGRepository) CompletePasswordReset(ctx context.Context, userID, resetID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, resetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryResetGone
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL AND ($2 = '' OR id::text <> $2)
	`, userID, keepSessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

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
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
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
