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
	ErrRepositoryNotFound    = errors.New("auth record not found")
)

type StoredUserWithPassword struct {
	User         User
	PasswordHash string
}

type StoredResetCode struct {
	UserID    string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (string, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (sessionID, userID string, err error)
	RotateSessionToken(ctx context.Context, sessionID, currentHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) error
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	UpsertResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetResetCode(ctx context.Context, userID string) (StoredResetCode, error)
	ConsumeResetAttempt(ctx context.Context, userID string, maxAttempts int) (codeHash string, err error)
	ResetPassword(ctx context.Context, userID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// CreateUser inserts the account together with its empty profile and default
// preferences so every user always has both rows.
func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var user User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, role, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrRepositoryEmailExists
		}
		return User{}, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO profiles (user_id) VALUES ($1)`, user.ID); err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1)`, user.ID); err != nil {
		return User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `WHERE email = $1`, email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `WHERE id = $1`, userID)
}

func (r *PGRepository) getUserAuth(ctx context.Context, where string, arg string) (StoredUserWithPassword, error) {
	var stored StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, role, suspended_at IS NOT NULL, created_at, password_hash
		FROM users
		`+where, arg).Scan(&stored.User.ID, &stored.User.Email, &stored.User.Role, &stored.User.Suspended, &stored.User.CreatedAt, &stored.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return stored, nil
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (string, error) {
	var sessionID string
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`, userID, tokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt).Scan(&sessionID)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (string, string, error) {
	var sessionID, userID string
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id
		FROM sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`, tokenHash).Scan(&sessionID, &userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrRepositoryNotFound
		}
		return "", "", err
	}
	return sessionID, userID, nil
}

// RotateSessionToken swaps the refresh token hash atomically; the current hash
// in the WHERE clause makes concurrent refreshes with the same token fail.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = NOW()
		WHERE id = $1
		  AND token_hash = $2
		  AND revoked_at IS NULL
	`, sessionID, currentHash, nextTokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash)
	return err
}

// SessionActive also refreshes the user's activity timestamp at most every
// five minutes, which feeds discovery ranking without a write per request.
func (r *PGRepository) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.id = $1 AND s.user_id = $2 AND s.revoked_at IS NULL AND s.expires_at > NOW()
			  AND u.suspended_at IS NULL
		)
	`, sessionID, userID).Scan(&active)
	if err != nil || !active {
		return active, err
	}
	_, err = r.dbPool.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return true, err
}

func (r *PGRepository) UpsertResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO password_reset_codes (user_id, code_hash, attempts, expires_at, created_at)
		VALUES ($1, $2, 0, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET code_hash = EXCLUDED.code_hash, attempts = 0, expires_at = EXCLUDED.expires_at, created_at = NOW()
	`, userID, codeHash, expiresAt)
	return err
}

func (r *PGRepository) GetResetCode(ctx context.Context, userID string) (StoredResetCode, error) {
	var code StoredResetCode
	err := r.dbPool.QueryRow(ctx, `
		SELECT user_id, code_hash, attempts, expires_at, created_at
		FROM password_reset_codes
		WHERE user_id = $1
	`, userID).Scan(&code.UserID, &code.CodeHash, &code.Attempts, &code.ExpiresAt, &code.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredResetCode{}, ErrRepositoryNotFound
		}
		return StoredResetCode{}, err
	}
	return code, nil
}

// ConsumeResetAttempt atomically counts one verification attempt and returns
// the code hash, or ErrRepositoryNotFound when there is no usable code (none,
// expired, or attempts exhausted).
func (r *PGRepository) ConsumeResetAttempt(ctx context.Context, userID string, maxAttempts int) (string, error) {
	var codeHash string
	err := r.dbPool.QueryRow(ctx, `
		UPDATE password_reset_codes SET attempts = attempts + 1
		WHERE user_id = $1 AND attempts < $2 AND expires_at > NOW()
		RETURNING code_hash
	`, userID, maxAttempts).Scan(&codeHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return codeHash, err
}

// ResetPassword updates the hash, consumes the code, revokes every session
// and forgets the account's push tokens.
func (r *PGRepository) ResetPassword(ctx context.Context, userID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM password_reset_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	// Every device is signed out: stop pushing this account's activity to them.
	if _, err := tx.Exec(ctx, `DELETE FROM push_tokens WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteUser removes the account; foreign keys cascade to profile, photos,
// swipes, matches, conversations, messages, blocks, notifications, sessions.
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

func nullableString(value string) any {
	if value == "" {
		return nil
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

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return strings.ToValidUTF8(value[:max], "")
}
