package auth

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
	"unicode/utf8"

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
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentHash, nextHash, userAgent, ipAddress string, expiresAt time.Time) error
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
	TouchUser(ctx context.Context, userID string) error
	CountRecentResets(ctx context.Context, userID string, since time.Time) (int, error)
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetLatestActiveReset(ctx context.Context, userID string) (StoredReset, error)
	IncrementResetAttempts(ctx context.Context, resetID string) (int, error)
	CompletePasswordReset(ctx context.Context, userID, resetID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
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

func (r *PGRepository) getUser(ctx context.Context, where string, arg any) (StoredUser, error) {
	var user StoredUser
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE `+where+` = $1
	`, arg).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUser{}, ErrRepositoryNotFound
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (StoredUser, error) {
	return r.getUser(ctx, "email", email)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	return r.getUser(ctx, "id", userID)
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID)
	return session, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id
		FROM sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`, tokenHash).Scan(&session.ID, &session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositoryNotFound
		}
		return StoredSession{}, err
	}
	return session, nil
}

// RotateSessionToken swaps the refresh token hash only if it still matches
// the presented one, so a refresh token can be used exactly once.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentHash, nextHash, userAgent, ipAddress string, expiresAt time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = NOW()
		WHERE id = $1
		  AND token_hash = $2
		  AND revoked_at IS NULL
	`, sessionID, currentHash, nextHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.db.QueryRow(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
		RETURNING id, user_id
	`, tokenHash).Scan(&session.ID, &session.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositoryNotFound
	}
	return session, err
}

func (r *PGRepository) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	var active bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&active)
	return active, err
}

func (r *PGRepository) TouchUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *PGRepository) CountRecentResets(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM password_resets WHERE user_id = $1 AND created_at >= $2
	`, userID, since).Scan(&count)
	return count, err
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Only the most recent code stays valid.
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

func (r *PGRepository) GetLatestActiveReset(ctx context.Context, userID string) (StoredReset, error) {
	var reset StoredReset
	err := r.db.QueryRow(ctx, `
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

func (r *PGRepository) IncrementResetAttempts(ctx context.Context, resetID string) (int, error) {
	var attempts int
	err := r.db.QueryRow(ctx, `
		UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1 RETURNING attempts
	`, resetID).Scan(&attempts)
	return attempts, err
}

func (r *PGRepository) CompletePasswordReset(ctx context.Context, userID, resetID, passwordHash string) error {
	tx, err := r.db.Begin(ctx)
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

// DeleteUser removes the account. Foreign keys cascade to profile, photos,
// swipes, matches, conversations, messages, blocks, sessions and
// notifications. Reports keep their moderation record with a NULL author.
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// truncate cuts to at most max bytes without splitting a UTF-8 sequence.
func truncate(value string, max int) string {
	value = strings.ToValidUTF8(value, "")
	if len(value) <= max {
		return value
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut]
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
