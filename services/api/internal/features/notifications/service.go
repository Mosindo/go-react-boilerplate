package notifications

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrNotFound      = errors.New("notification not found")
	ErrInvalidCursor = errors.New("invalid cursor")
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string) (Page, error) {
	if limit < 1 || limit > httpx.MaxLimit {
		limit = httpx.DefaultLimit
	}
	var before *Cursor
	if cursor != "" {
		c, err := decodeCursor(cursor)
		if err != nil {
			return Page{}, ErrInvalidCursor
		}
		before = &c
	}
	items, next, err := s.repo.List(ctx, userID, limit, before)
	if err != nil {
		return Page{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	page := Page{Notifications: items, UnreadCount: unread}
	if next != nil {
		page.NextCursor = encodeCursor(*next)
	}
	return page, nil
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

// Matched implements matching.Notifier. Failures are logged, never propagated:
// the match itself is already committed.
func (s *Service) Matched(ctx context.Context, matchID, conversationID, userA, userB string) {
	for _, pair := range [][2]string{{userA, userB}, {userB, userA}} {
		recipient, actor := pair[0], pair[1]
		if err := s.repo.Insert(ctx, recipient, TypeMatch, actor, conversationID); err != nil {
			log.Printf(`{"event":"notify_error","type":"match","error":%q}`, err.Error())
		}
		s.pub.Publish(recipient, "match", map[string]string{
			"matchId": matchID, "conversationId": conversationID, "userId": actor,
		})
	}
}

// NewMessage implements chat.Notifier.
func (s *Service) NewMessage(ctx context.Context, recipientID, senderID, conversationID string) {
	if err := s.repo.Insert(ctx, recipientID, TypeMessage, senderID, conversationID); err != nil {
		log.Printf(`{"event":"notify_error","type":"message","error":%q}`, err.Error())
	}
}

func encodeCursor(c Cursor) string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID))
}

func decodeCursor(raw string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Cursor{}, err
	}
	ts, id, ok := strings.Cut(string(b), "|")
	if !ok || !httpx.IsUUID(id) {
		return Cursor{}, ErrInvalidCursor
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return Cursor{}, err
	}
	return Cursor{CreatedAt: t, ID: id}, nil
}
