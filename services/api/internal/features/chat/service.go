package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/events"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/validate"
)

type Service struct {
	repo Repository
	pub  events.Publisher
}

func NewService(repo Repository, pub events.Publisher) *Service {
	if pub == nil {
		pub = events.Nop{}
	}
	return &Service{repo: repo, pub: pub}
}

// authorize returns the participant's access. Non-participants and blocked pairs
// both look like "not found" to readers.
func (s *Service) authorize(ctx context.Context, rawID, userID string) (Access, error) {
	id, ok := httpx.NormalizeUUID(rawID)
	if !ok {
		return Access{}, httpx.NotFound("conversation not found")
	}
	acc, err := s.repo.Access(ctx, id, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Access{}, httpx.NotFound("conversation not found")
		}
		return Access{}, err
	}
	return acc, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID, rawCursor string, limit int) (httpx.Page[Message], error) {
	acc, err := s.authorize(ctx, conversationID, userID)
	if err != nil {
		return httpx.Page[Message]{}, err
	}
	if acc.Blocked {
		return httpx.Page[Message]{}, httpx.NotFound("conversation not found")
	}
	var cursor *Cursor
	if rawCursor != "" {
		var c Cursor
		if err := httpx.DecodeCursor(rawCursor, &c); err != nil {
			return httpx.Page[Message]{}, err
		}
		if !httpx.IsUUID(c.ID) || c.CreatedAt.IsZero() {
			return httpx.Page[Message]{}, httpx.BadRequest("invalid cursor")
		}
		cursor = &c
	}
	items, err := s.repo.ListMessages(ctx, acc.ConversationID, cursor, limit+1)
	if err != nil {
		return httpx.Page[Message]{}, err
	}
	page := httpx.Page[Message]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		next := httpx.EncodeCursor(Cursor{CreatedAt: last.CreatedAt.Truncate(time.Microsecond), ID: last.ID})
		page.NextCursor = &next
	}
	if page.Items == nil {
		page.Items = []Message{}
	}
	return page, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	id, ok := httpx.NormalizeUUID(conversationID)
	if !ok {
		return Message{}, httpx.NotFound("conversation not found")
	}
	clean, err := validate.Text(body, 1, MaxBodyLen, true)
	if err != nil {
		return Message{}, httpx.BadRequest("body must be 1 to 2000 characters")
	}
	res, err := s.repo.Send(ctx, id, userID, clean)
	switch {
	case errors.Is(err, ErrNotFound):
		return Message{}, httpx.NotFound("conversation not found")
	case errors.Is(err, ErrBlocked):
		return Message{}, httpx.Blocked()
	case err != nil:
		return Message{}, err
	}
	s.pub.Publish(userID, events.MessageNew, res.Message)
	s.pub.Publish(res.Recipient, events.MessageNew, res.Message)
	s.pub.Publish(res.Recipient, events.NotificationNew, res.Notification)
	return res.Message, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	acc, err := s.authorize(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if acc.Blocked {
		return httpx.NotFound("conversation not found")
	}
	n, err := s.repo.MarkRead(ctx, acc.ConversationID, userID)
	if err != nil {
		return err
	}
	if n > 0 {
		s.pub.Publish(acc.OtherUserID, events.MessagesRead, ReadEvent{ConversationID: acc.ConversationID, ReaderID: userID})
	}
	return nil
}
