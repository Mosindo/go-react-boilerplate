package comments

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCommentPostNotFound = errors.New("comment post not found")
)

type StoredComment struct {
	ID           string
	PostID       string
	AuthorUserID string
	Content      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	ListCommentsByPost(ctx context.Context, organizationID, postID string, limit, offset int) ([]StoredComment, error)
	CreateComment(ctx context.Context, organizationID, postID, authorUserID, content string) (StoredComment, error)
	PostExists(ctx context.Context, organizationID, postID string) (bool, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ListCommentsByPost(ctx context.Context, organizationID, postID string, limit, offset int) ([]StoredComment, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT c.id, c.post_id, c.author_user_id, c.content, c.created_at, c.updated_at
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		JOIN users u ON u.id = p.author_user_id
		WHERE u.organization_id = $1
		  AND c.post_id = $2
		ORDER BY created_at ASC, id ASC
		LIMIT $3 OFFSET $4
	`, organizationID, postID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]StoredComment, 0)
	for rows.Next() {
		var comment StoredComment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.AuthorUserID, &comment.Content, &comment.CreatedAt, &comment.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return comments, nil
}

func (r *PGRepository) CreateComment(ctx context.Context, organizationID, postID, authorUserID, content string) (StoredComment, error) {
	var comment StoredComment
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO comments (post_id, author_user_id, content)
		SELECT p.id, u.id, $4
		FROM posts p
		JOIN users post_owner ON post_owner.id = p.author_user_id
		JOIN users u ON u.id = $2
		WHERE p.id = $1
		  AND post_owner.organization_id = $3
		  AND u.organization_id = $3
		RETURNING id, post_id, author_user_id, content, created_at, updated_at
	`, postID, authorUserID, organizationID, content).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.AuthorUserID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return StoredComment{}, ErrCommentPostNotFound
		}
		return StoredComment{}, err
	}
	return comment, nil
}

func (r *PGRepository) PostExists(ctx context.Context, organizationID, postID string) (bool, error) {
	var exists bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM posts p
			JOIN users u ON u.id = p.author_user_id
			WHERE p.id = $1
			  AND u.organization_id = $2
		)
	`, postID, organizationID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
