package auth

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryEmailExists = errors.New("auth email already exists")
	ErrRepositoryNotFound    = errors.New("auth user not found")
	ErrRepositorySessionGone = errors.New("auth session not found")
	ErrRepositoryResetToken  = errors.New("auth reset token invalid")
)

type StoredUser struct {
	ID    string
	Email string
}

type StoredUserWithPassword struct {
	User         StoredUser
	PasswordHash string
}

type StoredSession struct {
	ID     string
	UserID string
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string, birthDate time.Time) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	TouchUser(ctx context.Context, userID string) error
	LoadMe(ctx context.Context, userID string, now time.Time) (MeResponse, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	// CreateResetToken stores a hashed token; it reports false when a token was issued for the
	// user very recently (mail-bombing protection) and nothing was stored.
	CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (bool, error)
	// ConsumeResetToken atomically marks the token used, sets the new password hash and revokes
	// all sessions of the user.
	ConsumeResetToken(ctx context.Context, tokenHash, newPasswordHash string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string, birthDate time.Time) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, birth_date)
		VALUES ($1, $2, $3)
		RETURNING id, email
	`, email, passwordHash, birthDate).Scan(&user.ID, &user.Email)
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
	err := r.dbPool.QueryRow(ctx, `SELECT id, email, password_hash FROM users WHERE email = $1`, email).
		Scan(&user.User.ID, &user.User.Email, &user.PasswordHash)
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
	err := r.dbPool.QueryRow(ctx, `SELECT id, email FROM users WHERE id = $1`, userID).Scan(&user.ID, &user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUser{}, ErrRepositoryNotFound
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) LoadMe(ctx context.Context, userID string, now time.Time) (MeResponse, error) {
	return LoadMe(ctx, r.dbPool, userID, now)
}

func (r *PGRepository) TouchUser(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 512)), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID)
	if err != nil {
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&session.ID, &session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

// RotateSessionToken swaps the refresh token hash only if the presented one is still current,
// so two concurrent refreshes with the same token cannot both succeed.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = NOW()
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id
	`, sessionID, currentTokenHash, nextTokenHash, nullableString(truncate(userAgent, 512)), nullableIP(ipAddress), expiresAt).
		Scan(&session.ID, &session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW(), last_used_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (bool, error) {
	tag, err := r.dbPool.Exec(ctx, `
		INSERT INTO password_resets (user_id, token_hash, expires_at)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (
			SELECT 1 FROM password_resets
			WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 minute'
		)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PGRepository) ConsumeResetToken(ctx context.Context, tokenHash, newPasswordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE password_resets SET used_at = NOW()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRepositoryResetToken
		}
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, newPasswordHash); err != nil {
		return err
	}
	// Any other outstanding reset link is void once the password changed.
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
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

func truncate(s string, n int) string {
	if len(s) > n {
		return strings.ToValidUTF8(s[:n], "")
	}
	return s
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
