package notifications

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var ErrInvalidCursor = errors.New("invalid cursor")

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func encodeCursor(n Notification) string {
	return base64.RawURLEncoding.EncodeToString([]byte(n.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + n.ID))
}

func decodeCursor(raw string) (*cursor, error) {
	if raw == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidCursor
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil || len(parts[1]) != 36 {
		return nil, ErrInvalidCursor
	}
	return &cursor{CreatedAt: ts, ID: parts[1]}, nil
}

func (s *Service) List(ctx context.Context, userID, rawCursor string, limit int) (ListResponse, error) {
	cur, err := decodeCursor(rawCursor)
	if err != nil {
		return ListResponse{}, err
	}
	items, err := s.repo.List(ctx, userID, cur, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{}
	if len(items) > limit {
		items = items[:limit]
		resp.NextCursor = encodeCursor(items[len(items)-1])
	}
	resp.Notifications = items
	if resp.UnreadCount, err = s.repo.UnreadCount(ctx, userID); err != nil {
		return ListResponse{}, err
	}
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
