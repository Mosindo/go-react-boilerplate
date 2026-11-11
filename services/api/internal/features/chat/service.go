package chat

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrEmptyMessage   = errors.New("empty message")
	ErrMessageTooLong = errors.New("message too long")
)

const maxMessageRunes = 2000

type CardProvider interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

type Notifier interface {
	Notify(ctx context.Context, in notifications.Input)
	ClearConversation(ctx context.Context, userID, matchID string)
}

type Service struct {
	repo   Repository
	cards  CardProvider
	notify Notifier
	pub    realtime.Publisher
}

func NewService(repo Repository, cards CardProvider, notify Notifier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notify: notify, pub: pub}
}

func normalizeBody(body string) (string, error) {
	cleaned := strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, body))
	if cleaned == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(cleaned) > maxMessageRunes {
		return "", ErrMessageTooLong
	}
	return cleaned, nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit int, cursor string) (ConversationsResponse, error) {
	after, err := DecodeCursor(cursor)
	if err != nil {
		return ConversationsResponse{}, err
	}
	rows, err := s.repo.ListConversations(ctx, userID, limit+1, after)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{Conversations: make([]Conversation, 0, len(rows))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		resp.NextCursor = Cursor{At: last.ActivityAt, ID: last.MatchID}.Encode()
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherUserID)
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return ConversationsResponse{}, err
	}
	for _, r := range rows {
		card, ok := cards[r.OtherUserID]
		if !ok {
			continue
		}
		resp.Conversations = append(resp.Conversations, Conversation{
			MatchID: r.MatchID, CreatedAt: r.CreatedAt, User: card, LastMessage: r.LastMessage,
			UnreadCount: r.UnreadCount, OtherLastReadAt: r.OtherLastReadAt,
		})
	}
	return resp, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, matchID string, limit int, cursor string) (MessagesResponse, error) {
	before, err := DecodeCursor(cursor)
	if err != nil {
		return MessagesResponse{}, err
	}
	_, otherReadAt, err := s.repo.OtherMember(ctx, matchID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	msgs, err := s.repo.ListMessages(ctx, matchID, limit+1, before)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{OtherLastReadAt: otherReadAt}
	if len(msgs) > limit {
		msgs = msgs[:limit]
		oldest := msgs[len(msgs)-1]
		resp.NextCursor = Cursor{At: oldest.CreatedAt, ID: oldest.ID}.Encode()
	}
	// Reverse to chronological order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	resp.Messages = msgs
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, body string) (Message, error) {
	cleaned, err := normalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	msg, recipient, err := s.repo.InsertMessage(ctx, matchID, userID, cleaned)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{Type: realtime.EventMessageCreated, Data: MessageEvent{Message: msg}}
	s.pub.Publish(recipient, event)
	s.pub.Publish(userID, event)

	name, err := s.repo.FirstName(ctx, userID)
	if err != nil {
		name = "Ton match"
	}
	preview := cleaned
	if utf8.RuneCountInString(preview) > 80 {
		preview = string([]rune(preview)[:80]) + "…"
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID:    recipient,
		Type:      notifications.TypeMessage,
		Title:     "Nouveau message de " + name,
		Body:      preview,
		Data:      map[string]string{"matchId": matchID},
		DedupeKey: notifications.MessageDedupeKey(matchID),
	})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) error {
	other, _, err := s.repo.OtherMember(ctx, matchID, userID)
	if err != nil {
		return err
	}
	at, err := s.repo.MarkRead(ctx, matchID, userID)
	if err != nil {
		return err
	}
	s.notify.ClearConversation(ctx, userID, matchID)
	s.pub.Publish(other, realtime.Event{Type: realtime.EventConversationRead, Data: ReadEvent{MatchID: matchID, UserID: userID, ReadAt: at}})
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, matchID string) error {
	if err := s.repo.Hide(ctx, matchID, userID); err != nil {
		return err
	}
	s.notify.ClearConversation(ctx, userID, matchID)
	return nil
}
