package posts

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrTitleRequired = errors.New("post title required")
	ErrBodyRequired  = errors.New("post body required")
	ErrTitleTooLong  = errors.New("post title too long")
	ErrBodyTooLong   = errors.New("post body too long")
)

type Service struct {
	repo Repository
}

const (
	defaultPostsLimit = 20
	maxPostsLimit     = 100
	maxPostTitleRunes = 160
	maxPostBodyRunes  = 10000
)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListWithPagination(ctx context.Context, organizationID string, limit, offset int) ([]Post, error) {
	normalizedLimit := normalizePostsLimit(limit)
	normalizedOffset := normalizePostsOffset(offset)
	posts, err := s.repo.ListPosts(ctx, organizationID, normalizedLimit, normalizedOffset)
	if err != nil {
		return nil, err
	}

	payload := make([]Post, 0, len(posts))
	for _, post := range posts {
		payload = append(payload, mapStoredPost(post))
	}
	return payload, nil
}

func (s *Service) Create(ctx context.Context, organizationID, authorUserID, title, body string) (Post, error) {
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		return Post{}, ErrTitleRequired
	}
	if utf8.RuneCountInString(normalizedTitle) > maxPostTitleRunes {
		return Post{}, ErrTitleTooLong
	}

	normalizedBody := strings.TrimSpace(body)
	if normalizedBody == "" {
		return Post{}, ErrBodyRequired
	}
	if utf8.RuneCountInString(normalizedBody) > maxPostBodyRunes {
		return Post{}, ErrBodyTooLong
	}

	post, err := s.repo.CreatePost(ctx, organizationID, authorUserID, normalizedTitle, normalizedBody)
	if err != nil {
		return Post{}, err
	}

	return mapStoredPost(post), nil
}

func (s *Service) GetByID(ctx context.Context, organizationID, postID string) (Post, error) {
	post, err := s.repo.GetPostByID(ctx, organizationID, postID)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return Post{}, ErrPostNotFound
		}
		return Post{}, err
	}
	return mapStoredPost(post), nil
}

func mapStoredPost(post StoredPost) Post {
	return Post{
		ID:           post.ID,
		AuthorUserID: post.AuthorUserID,
		Title:        post.Title,
		Body:         post.Body,
		CreatedAt:    post.CreatedAt,
		UpdatedAt:    post.UpdatedAt,
	}
}

func normalizePostsLimit(limit int) int {
	if limit <= 0 {
		return defaultPostsLimit
	}
	if limit > maxPostsLimit {
		return maxPostsLimit
	}
	return limit
}

func normalizePostsOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
