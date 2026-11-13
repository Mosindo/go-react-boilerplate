package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrMessageContentNeeded = errors.New("message content required")
	ErrMessageTooLong       = errors.New("message must be at most 2000 characters")
	ErrSelfConversation     = errors.New("cannot open a conversation with yourself")
)

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string, dedupeKey string)
	MarkConversationRead(ctx context.Context, userID, peerUserID string)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	notifier  Notifier
	publisher realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, notifier Notifier, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.Nop{}
	}
	return &Service{repo: repo, cards: cards, notifier: notifier, publisher: publisher}
}

// NormalizeMessage trims and validates message content.
func NormalizeMessage(content string) (string, error) {
	normalized := strings.TrimSpace(content)
	if normalized == "" {
		return "", ErrMessageContentNeeded
	}
	if utf8.RuneCountInString(normalized) > maxMessageRunes {
		return "", ErrMessageTooLong
	}
	return normalized, nil
}

func (s *Service) authorize(ctx context.Context, userID, otherUserID string) error {
	if userID == otherUserID {
		return ErrSelfConversation
	}
	ok, err := s.repo.CanChat(ctx, userID, otherUserID)
	if err != nil {
		return err
	}
	if !ok {
		// Same error whether the user does not exist, never matched, was
		// unmatched or blocked: nothing leaks about the other member.
		return ErrConversationNotFound
	}
	return nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	if limit <= 0 || limit > maxChatsLimit {
		limit = defaultChatsLimit
	}
	stored, err := s.repo.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(stored))
	for _, c := range stored {
		ids = append(ids, c.PeerID)
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.PublicProfile, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}

	out := make([]Conversation, 0, len(stored))
	for _, c := range stored {
		card, ok := byID[c.PeerID]
		if !ok {
			continue
		}
		out = append(out, Conversation{
			MatchID:   c.MatchID,
			MatchedAt: c.MatchedAt,
			User: Peer{
				UserID:    card.UserID,
				FirstName: card.FirstName,
				Age:       card.Age,
				PhotoURL:  card.PrimaryPhotoURL(),
				IsDemo:    card.IsDemo,
			},
			LastMessage: c.LastMessage,
			UnreadCount: c.UnreadCount,
		})
	}
	return out, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, otherUserID, beforeID string, limit int) (MessagesPage, error) {
	if err := s.authorize(ctx, userID, otherUserID); err != nil {
		return MessagesPage{}, err
	}
	if limit <= 0 || limit > maxMessagesLimit {
		limit = defaultMessagesLimit
	}
	msgs, err := s.repo.ListMessages(ctx, userID, otherUserID, beforeID, limit+1)
	if err != nil {
		return MessagesPage{}, err
	}
	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}
	// Chronological order is friendlier for clients.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	peerRead, err := s.repo.PeerLastReadAt(ctx, userID, otherUserID)
	if err != nil {
		return MessagesPage{}, err
	}
	return MessagesPage{Messages: msgs, HasMore: hasMore, PeerLastReadAt: peerRead}, nil
}

func (s *Service) SendMessage(ctx context.Context, userID, otherUserID, content string) (Message, error) {
	normalized, err := NormalizeMessage(content)
	if err != nil {
		return Message{}, err
	}
	if err := s.authorize(ctx, userID, otherUserID); err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, userID, otherUserID, normalized)
	if err != nil {
		return Message{}, err
	}

	s.publisher.Publish(otherUserID, "message.created", msg)
	s.publisher.Publish(userID, "message.created", msg)
	if s.notifier != nil {
		name, _ := s.repo.FirstName(ctx, userID)
		if name == "" {
			name = "Votre match"
		}
		s.notifier.Notify(ctx, otherUserID, notifications.TypeMessage,
			"Nouveau message de "+name,
			"Ouvrez la conversation pour lui répondre.",
			map[string]string{"peerUserId": userID}, "peer:"+userID)
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, otherUserID string) (time.Time, error) {
	if err := s.authorize(ctx, userID, otherUserID); err != nil {
		return time.Time{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, userID, otherUserID)
	if err != nil {
		return time.Time{}, err
	}
	if s.notifier != nil {
		s.notifier.MarkConversationRead(ctx, userID, otherUserID)
	}
	s.publisher.Publish(otherUserID, "conversation.read", map[string]any{"userId": userID, "readAt": readAt})
	return readAt, nil
}

// Clear hides the current history for the caller only; the other member
// keeps their copy and the match stays active.
func (s *Service) Clear(ctx context.Context, userID, otherUserID string) error {
	if err := s.authorize(ctx, userID, otherUserID); err != nil {
		return err
	}
	return s.repo.Clear(ctx, userID, otherUserID)
}
