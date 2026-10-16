package chat

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
)

var ErrInvalidMessage = errors.New("message must contain between 1 and 2000 characters")

// Events lets the composition root fan messages out to realtime and notifications.
type Events interface {
	MessageCreated(ctx context.Context, m Message, recipientID string)
	MessagesRead(ctx context.Context, matchID, readerID, otherID string)
}

type ProfileSource interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileSource
	events   Events
}

func NewService(repo Repository, source ProfileSource, events Events) *Service {
	return &Service{repo: repo, profiles: source, events: events}
}

func (s *Service) Conversations(ctx context.Context, userID string, limit, offset int) ([]ConversationResponse, error) {
	stored, err := s.repo.Conversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(stored))
	for i, c := range stored {
		ids[i] = c.OtherID
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.PublicProfile, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	out := make([]ConversationResponse, 0, len(stored))
	for _, c := range stored {
		card, ok := byID[c.OtherID]
		if !ok {
			continue
		}
		out = append(out, ConversationResponse{
			MatchID: c.MatchID, CreatedAt: c.CreatedAt, ActivityAt: c.ActivityAt,
			Profile: card, LastMessage: c.LastMessage, UnreadCount: c.Unread,
		})
	}
	return out, nil
}

// Messages returns one page in chronological order. beforeID pages backwards.
func (s *Service) Messages(ctx context.Context, userID, matchID, beforeID string, limit int) (MessagesPage, error) {
	if _, err := s.repo.Participant(ctx, userID, matchID); err != nil {
		return MessagesPage{}, err
	}
	rows, err := s.repo.Messages(ctx, userID, matchID, beforeID, limit+1)
	if err != nil {
		return MessagesPage{}, err
	}
	page := MessagesPage{HasMore: len(rows) > limit}
	if page.HasMore {
		rows = rows[:limit]
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	if rows == nil {
		rows = []Message{}
	}
	page.Messages = rows
	return page, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, body string) (Message, error) {
	clean := cleanBody(body)
	if n := utf8.RuneCountInString(clean); n < 1 || n > MaxMessageRunes {
		return Message{}, ErrInvalidMessage
	}
	other, err := s.repo.Participant(ctx, userID, matchID)
	if err != nil {
		return Message{}, err
	}
	m, err := s.repo.Insert(ctx, matchID, userID, clean)
	if err != nil {
		return Message{}, err
	}
	s.events.MessageCreated(ctx, m, other)
	return m, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Participant(ctx, userID, matchID)
	if err != nil {
		return err
	}
	n, err := s.repo.MarkRead(ctx, userID, matchID)
	if err != nil {
		return err
	}
	if n > 0 {
		s.events.MessagesRead(ctx, matchID, userID, other)
	}
	return nil
}

func (s *Service) Clear(ctx context.Context, userID, matchID string) error {
	if _, err := s.repo.Participant(ctx, userID, matchID); err != nil {
		return err
	}
	return s.repo.Clear(ctx, userID, matchID)
}

func (s *Service) UnreadTotal(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadTotal(ctx, userID)
}

// cleanBody trims and removes control characters other than newlines/tabs.
func cleanBody(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r == '\n' || r == '\t' || (!unicode.IsControl(r) && r != utf8.RuneError) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
