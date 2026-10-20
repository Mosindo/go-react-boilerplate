package comments

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrPostNotFound          = errors.New("post not found")
	ErrCommentContentNeeded  = errors.New("comment content required")
	ErrCommentContentTooLong = errors.New("comment content too long")
)

type Service struct {
	repo Repository
}

const (
	defaultCommentsLimit   = 20
	maxCommentsLimit       = 200
	maxCommentContentRunes = 4000
)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByPost(ctx context.Context, organizationID, postID string, limit, offset int) ([]Comment, error) {
	exists, err := s.repo.PostExists(ctx, organizationID, postID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrPostNotFound
	}

	normalizedLimit := normalizeCommentsLimit(limit)
	normalizedOffset := normalizeCommentsOffset(offset)
	comments, err := s.repo.ListCommentsByPost(ctx, organizationID, postID, normalizedLimit, normalizedOffset)
	if err != nil {
		return nil, err
	}

	payload := make([]Comment, 0, len(comments))
	for _, comment := range comments {
		payload = append(payload, mapStoredComment(comment))
	}
	return payload, nil
}

func (s *Service) Create(ctx context.Context, organizationID, postID, authorUserID, content string) (Comment, error) {
	normalizedContent := strings.TrimSpace(content)
	if normalizedContent == "" {
		return Comment{}, ErrCommentContentNeeded
	}
	if utf8.RuneCountInString(normalizedContent) > maxCommentContentRunes {
		return Comment{}, ErrCommentContentTooLong
	}

	comment, err := s.repo.CreateComment(ctx, organizationID, postID, authorUserID, normalizedContent)
	if err != nil {
		if errors.Is(err, ErrCommentPostNotFound) {
			return Comment{}, ErrPostNotFound
		}
		return Comment{}, err
	}

	return mapStoredComment(comment), nil
}

func mapStoredComment(comment StoredComment) Comment {
	return Comment{
		ID:           comment.ID,
		PostID:       comment.PostID,
		AuthorUserID: comment.AuthorUserID,
		Content:      comment.Content,
		CreatedAt:    comment.CreatedAt,
		UpdatedAt:    comment.UpdatedAt,
	}
}

func normalizeCommentsLimit(limit int) int {
	if limit <= 0 {
		return defaultCommentsLimit
	}
	if limit > maxCommentsLimit {
		return maxCommentsLimit
	}
	return limit
}

func normalizeCommentsOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
