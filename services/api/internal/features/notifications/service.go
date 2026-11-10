package notifications

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/platform/cursor"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrNotFound      = errors.New("notification not found")
	ErrInvalidCursor = errors.New("invalid cursor")
)

const (
	DefaultLimit = 30
	MaxLimit     = 100
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

// NotifyMatch records and pushes a "new match" notification.
func (s *Service) NotifyMatch(ctx context.Context, userID, actorID, matchID, actorName string) {
	n, err := s.repo.Create(ctx, userID, TypeMatch, &actorID, &matchID, "Vous avez un nouveau match avec "+actorName+" !")
	s.publish(userID, n, err)
}

// NotifyMessage records (collapsed per conversation) and pushes a message notification.
func (s *Service) NotifyMessage(ctx context.Context, userID, actorID, matchID, actorName, preview string) {
	n, err := s.repo.UpsertMessage(ctx, userID, actorID, matchID, actorName+" : "+truncate(preview, 80))
	s.publish(userID, n, err)
}

func (s *Service) MarkMatchRead(ctx context.Context, userID, matchID string) error {
	return s.repo.MarkMatchRead(ctx, userID, matchID)
}

func (s *Service) publish(userID string, n Notification, err error) {
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","error":%q}`, err.Error())
		return
	}
	s.pub.Publish(userID, realtime.Event{Type: "notification.new", Data: ToResponse(n)})
}

type Page struct {
	Items       []Notification
	UnreadCount int
	NextCursor  *string
}

func (s *Service) List(ctx context.Context, userID, rawCursor string, limit int) (Page, error) {
	pos, err := cursor.Decode(rawCursor)
	if err != nil {
		return Page{}, ErrInvalidCursor
	}
	var before *time.Time
	beforeID := ""
	if pos != nil {
		before, beforeID = &pos.Time, pos.ID
	}
	items, err := s.repo.List(ctx, userID, before, beforeID, limit+1)
	if err != nil {
		return Page{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: items, UnreadCount: unread}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		next := cursor.Encode(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	return page, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func truncate(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}
