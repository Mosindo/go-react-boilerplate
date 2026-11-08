package notifications

import (
	"context"
	"errors"
	"log"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var ErrNotFound = platformerrors.ErrNotFound

// Notifier is what other features use to raise a notification.
type Notifier interface {
	Notify(ctx context.Context, userID, kind string, actorID, refID *string)
}

type Service struct {
	repo   Repository
	cards  profiles.CardLoader
	events realtime.Publisher
}

func NewService(repo Repository, cards profiles.CardLoader, events realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, events: events}
}

// Notify is best effort: a failure to notify must never fail the action that triggered it.
// In-app notifications are delivered live over the WebSocket and fetched with GET /notifications.
func (s *Service) Notify(ctx context.Context, userID, kind string, actorID, refID *string) {
	n, created, err := s.repo.Create(ctx, userID, kind, actorID, refID)
	if err != nil {
		log.Printf(`{"event":"notify_error","error":%q}`, err.Error())
		return
	}
	if !created {
		return
	}
	items, err := s.hydrate(ctx, userID, []stored{n})
	if err != nil || len(items) == 0 {
		return
	}
	unread, _ := s.repo.UnreadCount(ctx, userID)
	s.events.Publish(userID, realtime.Event{Type: "notification.new", Data: map[string]any{"notification": items[0], "unreadCount": unread}})
}

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) (ListResponse, error) {
	rows, err := s.repo.List(ctx, userID, before, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	var next *time.Time
	if len(rows) > limit {
		rows = rows[:limit]
		t := rows[len(rows)-1].CreatedAt
		next = &t
	}
	items, err := s.hydrate(ctx, userID, rows)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Notifications: items, UnreadCount: unread, NextBefore: next}, nil
}

func (s *Service) hydrate(ctx context.Context, userID string, rows []stored) ([]Notification, error) {
	ids := []string{}
	for _, r := range rows {
		if r.ActorID != nil {
			ids = append(ids, *r.ActorID)
		}
	}
	cards, err := s.cards.LoadCards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	actors := make(map[string]*Actor, len(cards))
	for _, c := range cards {
		a := &Actor{UserID: c.UserID, FirstName: c.FirstName}
		if len(c.Photos) > 0 {
			u := c.Photos[0].URL
			a.PhotoURL = &u
		}
		actors[c.UserID] = a
	}
	out := make([]Notification, 0, len(rows))
	for _, r := range rows {
		n := Notification{ID: r.ID, Type: r.Type, RefID: r.RefID, ReadAt: r.ReadAt, CreatedAt: r.CreatedAt}
		if r.ActorID != nil {
			n.Actor = actors[*r.ActorID]
		}
		out = append(out, n)
	}
	return out, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if !profiles.IsUUID(id) {
		return ErrNotFound
	}
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
