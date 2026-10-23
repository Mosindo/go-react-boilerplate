package notifications

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) create(ctx context.Context, userID, typ, title, body string, data Data) error {
	n, err := s.repo.Create(ctx, userID, typ, title, body, data)
	if err != nil {
		return err
	}
	s.pub.Publish(userID, "notification.new", n)
	return nil
}

// NotifyMatch tells userID that they matched with otherName.
func (s *Service) NotifyMatch(ctx context.Context, userID, matchID, otherUserID, otherName string) error {
	return s.create(ctx, userID, TypeMatch, "Nouveau match",
		"Vous avez un match avec "+otherName+" !", Data{MatchID: matchID, UserID: otherUserID})
}

// NotifyMessage tells recipientID about a new message (coalesced per conversation).
func (s *Service) NotifyMessage(ctx context.Context, recipientID, matchID, senderID, senderName, preview string) error {
	return s.create(ctx, recipientID, TypeMessage, "Message de "+senderName,
		truncateRunes(strings.TrimSpace(preview), 120), Data{MatchID: matchID, UserID: senderID})
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string) (ListResult, error) {
	var before *time.Time
	var beforeID string
	if cursor != "" {
		t, id, ok := pagination.Decode(cursor)
		if !ok {
			return ListResult{}, platformerrors.BadRequest("invalid_cursor", "invalid cursor")
		}
		before, beforeID = &t, id
	}
	items, err := s.repo.List(ctx, userID, limit+1, before, beforeID)
	if err != nil {
		return ListResult{}, err
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		c := pagination.Encode(last.CreatedAt, last.ID)
		next = &c
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Notifications: items, UnreadCount: unread, NextCursor: next}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return platformerrors.NotFound("notification_not_found", "notification not found")
		}
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// MarkConversationRead is called when a user opens a conversation.
func (s *Service) MarkConversationRead(ctx context.Context, userID, matchID string) error {
	return s.repo.MarkConversationRead(ctx, userID, matchID)
}
