package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrEmptyMessage = errors.New("chat: message is empty")
	ErrTooLong      = errors.New("chat: message too long")
)

type PhotoURLer interface{ URL(photoID string) string }

type Service struct {
	repo   Repository
	rt     realtime.Publisher
	photos PhotoURLer
}

func NewService(repo Repository, rt realtime.Publisher, photos PhotoURLer) *Service {
	return &Service{repo: repo, rt: rt, photos: photos}
}

func (s *Service) peer(id, name string, photoID *string) Peer {
	p := Peer{UserID: id, FirstName: name}
	if photoID != nil {
		u := s.photos.URL(*photoID)
		p.PhotoURL = &u
	}
	return p
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]ConversationSummary, error) {
	list, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]ConversationSummary, 0, len(list))
	for _, c := range list {
		out = append(out, ConversationSummary{
			ID: c.ID, MatchID: c.MatchID, Peer: s.peer(c.PeerID, c.PeerName, c.PhotoID),
			LastMessage: c.Last, UnreadCount: c.Unread, CreatedAt: c.CreatedAt, SortAt: c.SortAt,
		})
	}
	return out, nil
}

// Messages is membership-checked in SQL: a non-member gets ErrNotFound, never another user's data.
func (s *Service) Messages(ctx context.Context, userID, conversationID string, limit int, before *int64) (MessagesPage, error) {
	if _, err := s.repo.Membership(ctx, userID, conversationID); err != nil {
		return MessagesPage{}, err
	}
	desc, err := s.repo.Messages(ctx, userID, conversationID, limit+1, before)
	if err != nil {
		return MessagesPage{}, err
	}
	var next *int64
	if len(desc) > limit {
		desc = desc[:limit]
		id := desc[len(desc)-1].ID
		next = &id
	}
	for i, j := 0, len(desc)-1; i < j; i, j = i+1, j-1 {
		desc[i], desc[j] = desc[j], desc[i]
	}
	readAt, info, err := s.repo.PeerInfo(ctx, conversationID, userID)
	if err != nil {
		return MessagesPage{}, err
	}
	return MessagesPage{Messages: desc, NextBefore: next, PeerReadAt: readAt, Peer: s.peer(info.PeerID, info.PeerName, info.PhotoID)}, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Message{}, ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > MaxMessageRunes {
		return Message{}, ErrTooLong
	}
	peerID, err := s.repo.Membership(ctx, userID, conversationID)
	if err != nil {
		return Message{}, err
	}
	m, err := s.repo.Send(ctx, userID, conversationID, body)
	if err != nil {
		return Message{}, err
	}
	ev := realtime.Event{Type: "message", Data: m}
	s.rt.Publish(peerID, ev)
	s.rt.Publish(userID, ev) // keeps the sender's other devices in sync
	return m, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	peerID, err := s.repo.Membership(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	at, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	s.rt.Publish(peerID, realtime.Event{Type: "read", Data: map[string]any{"conversationId": conversationID, "readAt": at.Format(time.RFC3339Nano)}})
	return nil
}

func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	return s.repo.Clear(ctx, userID, conversationID)
}
