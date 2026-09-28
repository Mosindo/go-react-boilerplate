package matches

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
	now  func() time.Time
}

func NewService(repo Repository, pub events.Publisher) *Service {
	if pub == nil {
		pub = events.Nop{}
	}
	return &Service{repo: repo, pub: pub, now: time.Now}
}

// Finalize fills derived fields (age) of a stored row.
func Finalize(r Row, now time.Time) Summary {
	s := r.Summary
	s.User.Age = validate.AgeOn(r.BirthDate, now)
	return s
}

func (s *Service) List(ctx context.Context, viewerID, rawCursor string, limit int) (httpx.Page[Summary], error) {
	var cursor *Cursor
	if rawCursor != "" {
		var c Cursor
		if err := httpx.DecodeCursor(rawCursor, &c); err != nil {
			return httpx.Page[Summary]{}, err
		}
		if !httpx.IsUUID(c.ID) || c.SortAt.IsZero() {
			return httpx.Page[Summary]{}, httpx.BadRequest("invalid cursor")
		}
		cursor = &c
	}
	rows, err := s.repo.List(ctx, viewerID, cursor, limit+1)
	if err != nil {
		return httpx.Page[Summary]{}, err
	}
	page := httpx.Page[Summary]{Items: []Summary{}}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	now := s.now()
	for _, r := range rows {
		page.Items = append(page.Items, Finalize(r, now))
	}
	if more {
		last := rows[len(rows)-1]
		next := httpx.EncodeCursor(Cursor{SortAt: last.SortAt, ID: last.MatchID})
		page.NextCursor = &next
	}
	return page, nil
}

func (s *Service) Unmatch(ctx context.Context, viewerID, matchID string) error {
	id, ok := httpx.NormalizeUUID(matchID)
	if !ok {
		return httpx.NotFound("match not found")
	}
	other, ev, err := s.repo.Unmatch(ctx, viewerID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return httpx.NotFound("match not found")
		}
		return err
	}
	s.pub.Publish(viewerID, events.MatchRemoved, ev)
	s.pub.Publish(other, events.MatchRemoved, ev)
	return nil
}
