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
	ID             string
	Email          string
	OrganizationID string
	CreatedAt      time.Time
}

type StoredUserWithPassword struct {
	User         StoredUser
	PasswordHash string
	Suspended    bool
}

type StoredSession struct {
	ID             string
	UserID         string
	OrganizationID string
	ExpiresAt      time.Time
	CreatedAt      time.Time
	LastUsedAt     *time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetOpenPasswordReset(ctx context.Context, userID string) (StoredPasswordReset, error)
	IncrementPasswordResetAttempts(ctx context.Context, resetID string) error
	ConsumePasswordReset(ctx context.Context, resetID, userID, newPasswordHash string) error
}

type StoredPasswordReset struct {
	ID        string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
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
		INSERT INTO users (email, password_hash, organization_id)
		VALUES (
			$1,
			$2,
			(SELECT id FROM organizations WHERE slug = 'default')
		)
		RETURNING id, email, organization_id, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.OrganizationID, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	return user, nil
}

// IsSessionActive implements middleware.SessionValidator and, at most once a
// minute per user, refreshes users.last_active_at used by discovery ranking.
func (r *PGRepository) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.id = $1
			  AND s.user_id = $2
			  AND s.revoked_at IS NULL
			  AND s.expires_at > NOW()
			  AND u.status = 'active'
		)
	`, sessionID, userID).Scan(&active)
	if err != nil || !active {
		return false, err
	}
	_, _ = r.dbPool.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'
	`, userID)
	return true, nil
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetOpenPasswordReset(ctx context.Context, userID string) (StoredPasswordReset, error) {
	var reset StoredPasswordReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, code_hash, attempts, expires_at
		FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&reset.ID, &reset.CodeHash, &reset.Attempts, &reset.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredPasswordReset{}, ErrRepositoryNotFound
		}
		return StoredPasswordReset{}, err
	}
	return reset, nil
}

func (r *PGRepository) IncrementPasswordResetAttempts(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

// ConsumePasswordReset atomically marks the reset used, sets the new password
// and revokes every session of the user.
func (r *PGRepository) ConsumePasswordReset(ctx context.Context, resetID, userID, newPasswordHash string) error {
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
		return ErrRepositoryNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, newPasswordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	var user StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, organization_id, password_hash, created_at, status = 'suspended'
		FROM users
		WHERE email = $1
	`, email).Scan(&user.User.ID, &user.User.Email, &user.User.OrganizationID, &user.PasswordHash, &user.User.CreatedAt, &user.Suspended)
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
		SELECT id, email, organization_id, created_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.OrganizationID, &user.CreatedAt)
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

	user, err := r.GetUserByID(ctx, session.UserID)
	if err != nil {
		return StoredSession{}, err
	}
	session.OrganizationID = user.OrganizationID
	return session, nil
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT s.id, s.user_id, u.organization_id, s.expires_at, s.created_at, s.last_used_at
		FROM sessions s
		INNER JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
	`, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.OrganizationID,
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
		  AND expires_at > NOW()
		RETURNING id, user_id, expires_at, created_at, last_used_at
	`, sessionID, nextTokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt, usedAt, currentTokenHash).Scan(
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

	user, err := r.GetUserByID(ctx, session.UserID)
	if err != nil {
		return StoredSession{}, err
	}
	session.OrganizationID = user.OrganizationID
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
