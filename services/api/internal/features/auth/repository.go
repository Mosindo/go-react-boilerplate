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
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) (string, error)
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	GetPasswordHash(ctx context.Context, userID string) (string, error)
	CreateResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	CountRecentResetCodes(ctx context.Context, userID string, since time.Time) (int, error)
	ConsumeResetAttempt(ctx context.Context, userID string, maxAttempts int) (StoredResetCode, error)
	CompletePasswordReset(ctx context.Context, userID, codeID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) (DeletedUser, error)
}

type StoredResetCode struct {
	ID       string
	CodeHash string
	Attempts int
}

type DeletedUser struct {
	StorageKeys    []string
	MatchedUserIDs []string
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

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	var user StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, organization_id, password_hash, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(&user.User.ID, &user.User.Email, &user.User.OrganizationID, &user.PasswordHash, &user.User.CreatedAt)
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

// RotateSessionToken swaps the refresh token only if it is still the one the
// caller presented (compare-and-swap): two concurrent refreshes with the same
// token cannot both succeed.
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

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) (string, error) {
	var sessionID string
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET revoked_at = $2,
		    last_used_at = $2
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		RETURNING id
	`, tokenHash, revokedAt).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositorySessionGone
	}
	return sessionID, err
}

func (r *PGRepository) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&active)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" { // malformed uuid in claims
			return false, nil
		}
		return false, err
	}
	return active, nil
}

func (r *PGRepository) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	var hash string
	err := r.dbPool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return hash, err
}

// CreateResetCode invalidates previous codes and stores a new one.
func (r *PGRepository) CreateResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE password_reset_codes SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
	`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_reset_codes (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) CountRecentResetCodes(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	err := r.dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM password_reset_codes WHERE user_id = $1 AND created_at > $2
	`, userID, since).Scan(&count)
	return count, err
}

// ConsumeResetAttempt atomically spends one attempt on the active code and
// returns it. Concurrent guesses cannot exceed maxAttempts.
func (r *PGRepository) ConsumeResetAttempt(ctx context.Context, userID string, maxAttempts int) (StoredResetCode, error) {
	var code StoredResetCode
	err := r.dbPool.QueryRow(ctx, `
		UPDATE password_reset_codes
		SET attempts = attempts + 1
		WHERE id = (
			SELECT id FROM password_reset_codes
			WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
			ORDER BY created_at DESC
			LIMIT 1
		)
		  AND attempts < $2
		RETURNING id, code_hash, attempts::int
	`, userID, maxAttempts).Scan(&code.ID, &code.CodeHash, &code.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredResetCode{}, ErrRepositoryNotFound
	}
	return code, err
}

// CompletePasswordReset updates the password, consumes the code and revokes
// every session so a stolen session cannot survive a password reset.
func (r *PGRepository) CompletePasswordReset(ctx context.Context, userID, codeID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE password_reset_codes SET used_at = NOW()
		WHERE id = $1 AND user_id = $2 AND used_at IS NULL
	`, codeID, userID)
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

// DeleteUser removes the account. Every personal row (profile, preferences,
// photos, interests, swipes, matches, conversations membership, messages,
// notifications, sessions, blocks) is removed through ON DELETE CASCADE;
// reports keep their moderation record with the user reference set to NULL.
// It returns the photo storage keys to delete and the matched members to
// notify.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) (DeletedUser, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return DeletedUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	out := DeletedUser{StorageKeys: []string{}, MatchedUserIDs: []string{}}
	rows, err := tx.Query(ctx, `SELECT storage_key, thumb_storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return DeletedUser{}, err
	}
	for rows.Next() {
		var key, thumb string
		if err := rows.Scan(&key, &thumb); err != nil {
			rows.Close()
			return DeletedUser{}, err
		}
		out.StorageKeys = append(out.StorageKeys, key, thumb)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return DeletedUser{}, err
	}

	matchRows, err := tx.Query(ctx, `
		SELECT CASE WHEN user_low_id = $1 THEN user_high_id ELSE user_low_id END
		FROM matches WHERE user_low_id = $1 OR user_high_id = $1
	`, userID)
	if err != nil {
		return DeletedUser{}, err
	}
	for matchRows.Next() {
		var id string
		if err := matchRows.Scan(&id); err != nil {
			matchRows.Close()
			return DeletedUser{}, err
		}
		out.MatchedUserIDs = append(out.MatchedUserIDs, id)
	}
	matchRows.Close()
	if err := matchRows.Err(); err != nil {
		return DeletedUser{}, err
	}

	// Conversations whose membership becomes incomplete are deleted with
	// their messages (messages from the deleted member cascade anyway).
	if _, err := tx.Exec(ctx, `
		DELETE FROM conversations c
		USING conversation_participants cp
		WHERE cp.conversation_id = c.id AND cp.user_id = $1
	`, userID); err != nil {
		return DeletedUser{}, err
	}
	// Notifications held by other members that mention this account.
	if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE data->>'userId' = $1::text`, userID); err != nil {
		return DeletedUser{}, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return DeletedUser{}, err
	}
	if tag.RowsAffected() == 0 {
		return DeletedUser{}, ErrRepositoryNotFound
	}
	return out, tx.Commit(ctx)
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
