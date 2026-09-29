package notifications

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/realtime"
	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("notification not found")
	ErrBadRequest = errors.New("invalid request")
)

// Service persists notifications, pushes them in realtime and serves the REST reads.
// It implements notify.Notifier.
type Service struct {
	repo *Repository
	pub  realtime.Publisher
}

func NewService(repo *Repository, pub realtime.Publisher) *Service {
	if pub == nil {
		pub = realtime.NopPublisher{}
	}
	return &Service{repo: repo, pub: pub}
}

var _ notify.Notifier = (*Service)(nil)

func (s *Service) Notify(ctx context.Context, n notify.New) error {
	if n.UserID == "" || n.Type == "" {
		return ErrBadRequest
	}
	if n.Data == nil {
		n.Data = map[string]any{}
	}
	saved, err := s.repo.Upsert(ctx, n.UserID, n.Type, n.Title, n.Body, n.Data)
	if err != nil {
		return err
	}
	s.pub.Publish(n.UserID, realtime.Event{Type: "notification.new", Data: map[string]any{"notification": saved}})
	return nil
}

func (s *Service) List(ctx context.Context, userID, rawLimit, rawBefore, beforeID string) (ListResult, error) {
	limit, err := parseLimit(rawLimit)
	if err != nil {
		return ListResult{}, err
	}
	var before *time.Time
	if rawBefore != "" {
		t, err := time.Parse(time.RFC3339Nano, rawBefore)
		if err != nil {
			return ListResult{}, ErrBadRequest
		}
		before = &t
		if beforeID != "" {
			if _, err := uuid.Parse(beforeID); err != nil {
				return ListResult{}, ErrBadRequest
			}
		}
	} else {
		beforeID = ""
	}
	items, unread, err := s.repo.List(ctx, userID, before, beforeID, limit)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Notifications: items, UnreadCount: unread}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNotFound
	}
	ok, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func parseLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultLimit, nil
	}
	n := 0
	if len(raw) > 4 {
		return 0, ErrBadRequest
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, ErrBadRequest
		}
		n = n*10 + int(ch-'0')
	}
	if n < 1 || n > maxLimit {
		return 0, ErrBadRequest
	}
	return n, nil
}
