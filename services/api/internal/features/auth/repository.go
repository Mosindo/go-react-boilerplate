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
	ID              string
	Email           string
	CreatedAt       time.Time
	ProfileComplete bool
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
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	RevokeAllSessions(ctx context.Context, userID string, revokedAt time.Time) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string) (string, error)
	DeleteUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const userSelect = `
	SELECT u.id, u.email, u.created_at,
	       EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = u.id)
	       AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = u.id)
	FROM users u`

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, created_at, FALSE
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.ProfileComplete)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) getAuth(ctx context.Context, where string, arg string) (StoredUserWithPassword, error) {
	var u StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT u.id, u.email, u.created_at, u.password_hash,
		       EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = u.id)
		       AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = u.id)
		FROM users u WHERE `+where, arg).Scan(&u.User.ID, &u.User.Email, &u.User.CreatedAt, &u.PasswordHash, &u.User.ProfileComplete)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUserWithPassword{}, ErrRepositoryNotFound
	}
	return u, err
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	return r.getAuth(ctx, "u.email = $1", email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getAuth(ctx, "u.id = $1", userID)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, userSelect+` WHERE u.id = $1`, userID).
		Scan(&user.ID, &user.Email, &user.CreatedAt, &user.ProfileComplete)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUser{}, ErrRepositoryNotFound
	}
	return user, err
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	return s, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

// RotateSessionToken swaps the refresh token hash. It only succeeds for the exact token that was presented,
// so two concurrent refreshes with the same token cannot both win.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = $6
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id
	`, sessionID, nextTokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt, usedAt).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
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

func (r *PGRepository) RevokeAllSessions(ctx context.Context, userID string, revokedAt time.Time) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`, userID, revokedAt)
	return err
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	tag, err := r.dbPool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

// ConsumePasswordReset atomically marks a valid token used, sets the new password and revokes every session.
func (r *PGRepository) ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE password_resets SET used_at = NOW()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, newPasswordHash); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return "", err
	}
	return userID, tx.Commit(ctx)
}

// DeleteUser removes the account; every owned row (profile, photos, swipes, matches, messages, ...) cascades.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
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

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func nullableIP(value string) any {
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.String()
}
