package notifications

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/platform/signing"
)

var (
	ErrNotFound      = errors.New("notification not found")
	ErrInvalidCursor = errors.New("invalid cursor")
)

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

type Service struct {
	repo   Repository
	signer *signing.Signer
}

func NewService(repo Repository, signer *signing.Signer) *Service {
	return &Service{repo: repo, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string) (ListResponse, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	var beforeAt *time.Time
	var beforeID string
	if cursor != "" {
		at, id, err := decodeCursor(cursor)
		if err != nil {
			return ListResponse{}, ErrInvalidCursor
		}
		beforeAt, beforeID = &at, id
	}

	rows, err := s.repo.List(ctx, userID, limit+1, beforeAt, beforeID)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Notifications: make([]Notification, 0, len(rows))}
	if len(rows) > limit {
		last := rows[limit-1]
		resp.NextCursor = encodeCursor(last.CreatedAt, last.ID)
		rows = rows[:limit]
	}
	for _, rw := range rows {
		n := rw.Notification
		if n.Actor != nil && rw.actorPhotoID != nil {
			url := s.signer.PhotoURL(*rw.actorPhotoID)
			n.Actor.PhotoURL = &url
		}
		resp.Notifications = append(resp.Notifications, n)
	}
	resp.UnreadCount, err = s.repo.UnreadCount(ctx, userID)
	return resp, err
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func encodeCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(c string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return at, parts[1], nil
}
