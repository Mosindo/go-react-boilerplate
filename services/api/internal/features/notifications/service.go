package notifications

import (
	"context"
	"time"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
)

const maxMarkIDs = 100

type Service struct {
	repo   Repository
	signer *storage.Signer
}

func NewService(repo Repository, signer *storage.Signer) *Service {
	return &Service{repo: repo, signer: signer}
}

func (s *Service) List(ctx context.Context, userID, before string, limit int) (Page, error) {
	var cursor *time.Time
	if before != "" {
		t, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			return Page{}, perr.Invalid("invalid_cursor", "before must be an RFC 3339 timestamp")
		}
		cursor = &t
	}
	rows, err := s.repo.List(ctx, userID, cursor, limit+1)
	if err != nil {
		return Page{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: []Notification{}, UnreadCount: unread}
	if len(rows) > limit {
		rows = rows[:limit]
		next := rows[len(rows)-1].CreatedAt.Format(time.RFC3339Nano)
		page.NextBefore = &next
	}
	for _, r := range rows {
		page.Items = append(page.Items, Notification{
			ID: r.ID, Type: r.Type, ConversationID: r.ConversationID, ReadAt: r.ReadAt, CreatedAt: r.CreatedAt,
			Actor: storage.UserSummary{ID: r.ActorID, FirstName: r.ActorName, ThumbURL: s.signer.ThumbURL(r.ActorPhotoID)},
		})
	}
	return page, nil
}

func (s *Service) MarkRead(ctx context.Context, userID string, ids []string) error {
	if len(ids) > maxMarkIDs {
		return perr.Invalid("too_many_ids", "too many ids")
	}
	for _, id := range ids {
		if !isUUID(id) {
			return perr.Invalid("invalid_id", "ids must be notification ids")
		}
	}
	return s.repo.MarkRead(ctx, userID, ids)
}

func isUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, r := range v {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'):
			return false
		}
	}
	return true
}
