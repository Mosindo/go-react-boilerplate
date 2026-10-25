package notifications

import (
	"context"
	"log"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type SummaryReader interface {
	Summaries(ctx context.Context, userIDs []string) (map[string]profiles.ProfileSummary, error)
}

type Service struct {
	repo     *Repository
	profiles SummaryReader
	events   realtime.Publisher
}

func NewService(repo *Repository, profiles SummaryReader, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, events: events}
}

func (s *Service) hydrate(ctx context.Context, rows []row) ([]Notification, error) {
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.ActorID != nil {
			ids = append(ids, *r.ActorID)
		}
	}
	summaries, err := s.profiles.Summaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Notification, 0, len(rows))
	for _, r := range rows {
		n := Notification{ID: r.ID, Type: r.Type, ConversationID: r.ConversationID, ReadAt: r.ReadAt, CreatedAt: r.CreatedAt}
		if r.ActorID != nil {
			if sum, ok := summaries[*r.ActorID]; ok {
				n.Actor = &sum
			}
		}
		out = append(out, n)
	}
	return out, nil
}

func (s *Service) List(ctx context.Context, userID string, limit int, cur *httpx.Cursor) (ListResponse, error) {
	rows, unread, err := s.repo.list(ctx, userID, limit, cur)
	if err != nil {
		return ListResponse{}, err
	}
	page := httpx.Paginate(rows, limit, func(r row) httpx.Cursor { return httpx.Cursor{At: r.CreatedAt, ID: r.ID} })
	items, err := s.hydrate(ctx, page.Items)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Items: items, NextCursor: page.NextCursor, UnreadCount: unread}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.markRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.markAllRead(ctx, userID)
}

// Push sends a stored notification to the user's sockets. Failures are logged: the notification is
// already persisted and will be served by the REST API.
func (s *Service) Push(ctx context.Context, userID, notificationID string) {
	r, err := s.repo.get(ctx, userID, notificationID)
	if err != nil {
		log.Printf(`{"event":"notification_push_error","error":%q}`, err.Error())
		return
	}
	items, err := s.hydrate(ctx, []row{r})
	if err != nil || len(items) != 1 {
		return
	}
	s.events.Publish(userID, realtime.Event{Type: "notification", Data: items[0]})
}
