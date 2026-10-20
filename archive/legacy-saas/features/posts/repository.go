package posts

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPostNotFound = errors.New("post not found")
)

type StoredPost struct {
	ID           string
	AuthorUserID string
	Title        string
	Body         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	ListPosts(ctx context.Context, organizationID string, limit, offset int) ([]StoredPost, error)
	CreatePost(ctx context.Context, organizationID, authorUserID, title, body string) (StoredPost, error)
	GetPostByID(ctx context.Context, organizationID, postID string) (StoredPost, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ListPosts(ctx context.Context, organizationID string, limit, offset int) ([]StoredPost, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT p.id, p.author_user_id, p.title, p.body, p.created_at, p.updated_at
		FROM posts p
		JOIN users u ON u.id = p.author_user_id
		WHERE u.organization_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, organizationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]StoredPost, 0)
	for rows.Next() {
		var post StoredPost
		if err := rows.Scan(&post.ID, &post.AuthorUserID, &post.Title, &post.Body, &post.CreatedAt, &post.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return posts, nil
}

func (r *PGRepository) CreatePost(ctx context.Context, organizationID, authorUserID, title, body string) (StoredPost, error) {
	var post StoredPost
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO posts (author_user_id, title, body)
		SELECT u.id, $3, $4
		FROM users u
		WHERE u.id = $1
		  AND u.organization_id = $2
		RETURNING id, author_user_id, title, body, created_at, updated_at
	`, authorUserID, organizationID, title, body).Scan(
		&post.ID,
		&post.AuthorUserID,
		&post.Title,
		&post.Body,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		return StoredPost{}, err
	}
	return post, nil
}

func (r *PGRepository) GetPostByID(ctx context.Context, organizationID, postID string) (StoredPost, error) {
	var post StoredPost
	err := r.dbPool.QueryRow(ctx, `
		SELECT p.id, p.author_user_id, p.title, p.body, p.created_at, p.updated_at
		FROM posts p
		JOIN users u ON u.id = p.author_user_id
		WHERE u.organization_id = $1
		  AND p.id = $2
	`, organizationID, postID).Scan(
		&post.ID,
		&post.AuthorUserID,
		&post.Title,
		&post.Body,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredPost{}, ErrPostNotFound
		}
		return StoredPost{}, err
	}
	return post, nil
}
