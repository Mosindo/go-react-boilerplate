package chat

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/urlsign"
)

type Service struct {
	repo      *Repository
	signer    *urlsign.Signer
	publisher realtime.Publisher
	notifier  notify.Notifier
}

func NewService(repo *Repository, signer *urlsign.Signer, publisher realtime.Publisher, notifier notify.Notifier) *Service {
	return &Service{repo: repo, signer: signer, publisher: publisher, notifier: notifier}
}

// NormalizeBody trims the text, strips control characters (newlines and tabs are kept as
// newline/space) and enforces 1-2000 runes.
func NormalizeBody(raw string) (string, error) {
	if !utf8.ValidString(raw) {
		return "", ErrInvalidBody
	}
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		switch {
		case r == '\r':
			continue
		case r == '\n':
			b.WriteRune('\n')
		case r == '\t':
			b.WriteRune(' ')
		case unicode.IsControl(r), r == ' ', r == ' ', r == '‎' && false:
			continue
		default:
			b.WriteRune(r)
		}
	}
	body := strings.TrimSpace(b.String())
	n := utf8.RuneCountInString(body)
	if n < 1 || n > MaxBodyRunes {
		return "", ErrInvalidBody
	}
	return body, nil
}

// ClampLimit applies default and maximum page sizes.
func ClampLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

// truncateRunes shortens s to at most max runes, appending an ellipsis when cut.
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max-1])) + "…"
}

func (s *Service) ListConversations(ctx context.Context, userID string, before *time.Time, beforeID *string, limit int) ([]ConversationSummary, error) {
	rows, err := s.repo.ListConversations(ctx, userID, before, beforeID, ClampLimit(limit))
	if err != nil {
		return nil, err
	}
	out := make([]ConversationSummary, 0, len(rows))
	for _, r := range rows {
		cs := ConversationSummary{
			ID: r.ID, MatchID: r.MatchID, MatchedAt: r.MatchedAt, UpdatedAt: r.UpdatedAt,
			User:        ConversationUser{UserID: r.OtherID, FirstName: r.FirstName, Age: r.Age},
			LastMessage: r.LastMessage, UnreadCount: r.UnreadCount,
		}
		if r.PhotoID != nil {
			cs.User.Photo = &Photo{ID: *r.PhotoID, URL: s.signer.PhotoURL(*r.PhotoID), Position: r.PhotoPosition}
		}
		out = append(out, cs)
	}
	return out, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, beforeID *string, limit int) (*ListMessagesResponse, error) {
	other, err := s.repo.OtherParticipant(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	limit = ClampLimit(limit)
	msgs, err := s.repo.ListMessages(ctx, conversationID, userID, other, beforeID, limit)
	if err != nil {
		return nil, err
	}
	resp := &ListMessagesResponse{Messages: msgs}
	if len(msgs) > limit {
		resp.Messages = msgs[:limit]
		id := resp.Messages[limit-1].ID
		resp.NextCursor = &id
	}
	return resp, nil
}

func (s *Service) SendMessage(ctx context.Context, userID, conversationID, rawBody string) (*Message, error) {
	// Participation is proven (404) before any input detail is revealed.
	if _, err := s.repo.OtherParticipant(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	body, err := NormalizeBody(rawBody)
	if err != nil {
		return nil, err
	}
	res, err := s.repo.SendMessage(ctx, conversationID, userID, body)
	if err != nil {
		return nil, err
	}
	msg := res.Message
	event := realtime.Event{Type: "message.new", Data: map[string]any{"message": msg}}
	s.publisher.Publish(res.RecipientID, event)
	s.publisher.Publish(userID, event)

	preview := truncateRunes(strings.Join(strings.Fields(body), " "), notificationBodyMax)
	if res.SenderFirstName != "" {
		preview = res.SenderFirstName + ": " + preview
		preview = truncateRunes(preview, notificationBodyMax)
	}
	// Best effort: a failed notification must never fail a delivered message.
	_ = s.notifier.Notify(ctx, notify.New{
		UserID: res.RecipientID,
		Type:   notify.TypeMessage,
		Title:  "New message",
		Body:   preview,
		Data:   map[string]any{"conversationId": conversationID, "userId": userID},
	})
	return &msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	other, err := s.repo.OtherParticipant(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	readAt, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	s.publisher.Publish(other, realtime.Event{Type: "conversation.read", Data: map[string]any{
		"conversationId": conversationID, "readAt": readAt.UTC(),
	}})
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if _, err := s.repo.OtherParticipant(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.repo.Hide(ctx, conversationID, userID)
}
