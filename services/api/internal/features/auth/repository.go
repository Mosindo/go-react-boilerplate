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
	errRepoEmailExists = errors.New("auth email already exists")
	errRepoNotFound    = errors.New("auth record not found")
)

type storedUser struct {
	User
	PasswordHash     string
	ModerationStatus string
}

type storedSession struct {
	ID     string
	UserID string
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (storedUser, error)
	GetUserByID(ctx context.Context, userID string) (storedUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (storedSession, error)
	RotateSession(ctx context.Context, currentHash, nextHash, userAgent, ipAddress string, expiresAt time.Time) (storedSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	ValidateSession(ctx context.Context, sessionID, userID string) (bool, error)
	CountRecentResetTokens(ctx context.Context, userID string, since time.Time) (int, error)
	CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ConsumeResetToken(ctx context.Context, email, tokenHash, newPasswordHash string) (string, error)
	UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error
	DeleteUser(ctx context.Context, userID string) ([]string, error)
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
			return User{}, errRepoEmailExists
		}
		return User{}, err
	}
	return u, nil
}

func (r *PGRepository) getUser(ctx context.Context, where string, arg string) (storedUser, error) {
	var u storedUser
	err := r.db.QueryRow(ctx, `
		SELECT id, email, created_at, password_hash, moderation_status FROM users WHERE `+where+` = $1
	`, arg).Scan(&u.ID, &u.Email, &u.CreatedAt, &u.PasswordHash, &u.ModerationStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedUser{}, errRepoNotFound
	}
	return u, err
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (storedUser, error) {
	return r.getUser(ctx, "email", email)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (storedUser, error) {
	return r.getUser(ctx, "id", userID)
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (storedSession, error) {
	var s storedSession
	err := r.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	return s, err
}

// RotateSession atomically swaps the refresh token hash. A replayed (already rotated)
// refresh token matches no row and is rejected.
func (r *PGRepository) RotateSession(ctx context.Context, currentHash, nextHash, userAgent, ipAddress string, expiresAt time.Time) (storedSession, error) {
	var s storedSession
	err := r.db.QueryRow(ctx, `
		UPDATE sessions s
		SET token_hash = $2,
		    user_agent = COALESCE($3, s.user_agent),
		    ip_address = COALESCE($4, s.ip_address),
		    expires_at = $5,
		    last_used_at = NOW()
		FROM users u
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
		  AND u.id = s.user_id
		  AND u.moderation_status <> 'suspended'
		RETURNING s.id, s.user_id
	`, currentHash, nextHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedSession{}, errRepoNotFound
	}
	return s, err
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

// ValidateSession checks the session in a single round trip and refreshes the user's
// activity timestamp at most every five minutes (used for discovery ranking).
func (r *PGRepository) ValidateSession(ctx context.Context, sessionID, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		WITH valid AS (
			SELECT u.id
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.id = $1 AND s.user_id = $2
			  AND s.revoked_at IS NULL AND s.expires_at > NOW()
			  AND u.moderation_status <> 'suspended'
		), touch AS (
			UPDATE users SET last_active_at = NOW()
			WHERE id IN (SELECT id FROM valid) AND last_active_at < NOW() - INTERVAL '5 minutes'
		)
		SELECT EXISTS (SELECT 1 FROM valid)
	`, sessionID, userID).Scan(&ok)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" { // malformed uuid in claims
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (r *PGRepository) CountRecentResetTokens(ctx context.Context, userID string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = $1 AND created_at > $2
	`, userID, since).Scan(&n)
	return n, err
}

func (r *PGRepository) CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

// ConsumeResetToken marks the code used, sets the new password and revokes every session.
func (r *PGRepository) ConsumeResetToken(ctx context.Context, email, tokenHash, newPasswordHash string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE password_reset_tokens t
		SET used_at = NOW()
		FROM users u
		WHERE t.token_hash = $1 AND t.used_at IS NULL AND t.expires_at > NOW()
		  AND u.id = t.user_id AND u.email = $2
		RETURNING t.user_id
	`, tokenHash, email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errRepoNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, newPasswordHash); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return "", err
	}
	return userID, tx.Commit(ctx)
}

// UpdatePassword changes the hash and revokes every other session of the user.
func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL
	`, userID, keepSessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteUser removes the account; foreign keys cascade to profile, photos, swipes, matches,
// conversations, messages, blocks and notifications. Returns photo storage keys to purge.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) ([]string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, errRepoNotFound
	}
	return keys, tx.Commit(ctx)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	if len(value) > 512 {
		return value[:512]
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
