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
	ErrRepositoryEmailExists  = errors.New("auth email already exists")
	ErrRepositoryNotFound     = errors.New("auth record not found")
	ErrRepositorySessionGone  = errors.New("auth session not found")
	ErrRepositoryTooManyCodes = errors.New("too many reset codes requested")
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
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type StoredResetCode struct {
	ID       string
	CodeHash string
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, oldTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error
	CreateResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time, maxPerHour int) error
	UseResetCodeAttempt(ctx context.Context, userID string, maxAttempts int) (StoredResetCode, error)
	ConsumeResetCode(ctx context.Context, codeID, userID, passwordHash string) error
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
	return r.getUserAuth(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`, email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE id = $1`, userID)
}

func (r *PGRepository) getUserAuth(ctx context.Context, query string, arg string) (StoredUserWithPassword, error) {
	var user StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, query, arg).Scan(&user.User.ID, &user.User.Email, &user.PasswordHash, &user.User.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `SELECT id, email, created_at FROM users WHERE id = $1`, userID).
		Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
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
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id, expires_at, created_at
	`, userID, tokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).
		Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt)
	return session, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id, expires_at, created_at
		FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

// RotateSessionToken swaps the refresh token hash only if the presented one is
// still current, so a refresh token can be used exactly once even under races.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, oldTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = $7
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL
		RETURNING id, user_id, expires_at, created_at
	`, sessionID, oldTokenHash, nextTokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt, usedAt).
		Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	_, _ = r.dbPool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, session.UserID)
	return session, nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2, last_used_at = $2
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

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE sessions SET revoked_at = NOW()
			WHERE user_id = $1 AND revoked_at IS NULL AND id::text <> $2
		`, userID, keepSessionID)
		return err
	})
}

// CreateResetCode invalidates previous codes and stores the new one, unless the
// user already requested maxPerHour codes in the last hour.
func (r *PGRepository) CreateResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time, maxPerHour int) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		var recent int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM password_reset_codes
			WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 hour'
		`, userID).Scan(&recent); err != nil {
			return err
		}
		if recent >= maxPerHour {
			return ErrRepositoryTooManyCodes
		}
		if _, err := tx.Exec(ctx, `UPDATE password_reset_codes SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO password_reset_codes (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
		`, userID, codeHash, expiresAt)
		return err
	})
}

// UseResetCodeAttempt atomically counts one guess against the user's current
// code and returns it; it fails when the code is missing, expired, used or
// already guessed maxAttempts times.
func (r *PGRepository) UseResetCodeAttempt(ctx context.Context, userID string, maxAttempts int) (StoredResetCode, error) {
	var code StoredResetCode
	err := r.dbPool.QueryRow(ctx, `
		UPDATE password_reset_codes SET attempts = attempts + 1
		WHERE id = (
			SELECT id FROM password_reset_codes
			WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW() AND attempts < $2
			ORDER BY created_at DESC LIMIT 1
		)
		RETURNING id, code_hash
	`, userID, maxAttempts).Scan(&code.ID, &code.CodeHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredResetCode{}, ErrRepositoryNotFound
		}
		return StoredResetCode{}, err
	}
	return code, nil
}

func (r *PGRepository) ConsumeResetCode(ctx context.Context, codeID, userID, passwordHash string) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE password_reset_codes SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, codeID)
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

func (r *PGRepository) inTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
