package matches

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
	"github.com/google/uuid"
)

var (
	ErrInvalidAction = errors.New("action must be like or pass")
	ErrSelfSwipe     = errors.New("you cannot swipe yourself")
	ErrNotEligible   = errors.New("this profile is not available")
	ErrNotFound      = errors.New("match not found")
	ErrBadCursor     = errors.New("invalid cursor")
)

type CardSource interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.Card, error)
}

type Service struct {
	repo   Repository
	cards  CardSource
	events realtime.Publisher
}

func NewService(repo Repository, cards CardSource, events realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, events: events}
}

func (s *Service) Swipe(ctx context.Context, me string, req SwipeRequest) (SwipeResult, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	target := strings.ToLower(strings.TrimSpace(req.TargetID))
	if _, err := uuid.Parse(target); err != nil {
		return SwipeResult{}, ErrNotEligible
	}
	if target == me {
		return SwipeResult{}, ErrSelfSwipe
	}
	out, err := s.repo.Swipe(ctx, me, target, action)
	if err != nil {
		if errors.Is(err, ErrRepoNotEligible) {
			return SwipeResult{}, ErrNotEligible
		}
		return SwipeResult{}, err
	}
	res := SwipeResult{Action: out.Action, Matched: out.Matched, MatchID: out.MatchID}
	if out.Matched {
		cards, err := s.cards.Cards(ctx, me, []string{target})
		if err == nil && len(cards) == 1 {
			res.User = &cards[0]
		}
	}
	if out.NewMatch {
		ev := realtime.Event{Type: "match", Payload: map[string]string{"matchId": out.MatchID}}
		s.events.Publish(me, ev)
		s.events.Publish(target, ev)
	}
	return res, nil
}

func (s *Service) List(ctx context.Context, me, cursor string, limit int) (MatchPage, error) {
	after, err := decodeCursor(cursor)
	if err != nil {
		return MatchPage{}, err
	}
	rows, err := s.repo.List(ctx, me, after, limit+1)
	if err != nil {
		return MatchPage{}, err
	}
	page := MatchPage{Matches: []MatchItem{}}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		page.NextCursor = encodeCursor(Cursor{SortAt: last.SortAt, ID: last.ID})
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.OtherID
	}
	cards, err := s.cards.Cards(ctx, me, ids)
	if err != nil {
		return MatchPage{}, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	for _, r := range rows {
		card, ok := byID[r.OtherID]
		if !ok {
			continue
		}
		item := MatchItem{ID: r.ID, CreatedAt: r.CreatedAt, User: card, UnreadCount: r.UnreadCount}
		if r.LastBody != nil && r.LastSender != nil && r.LastAt != nil {
			item.LastMessage = &LastMessage{Body: *r.LastBody, SenderID: *r.LastSender, CreatedAt: *r.LastAt}
		}
		page.Matches = append(page.Matches, item)
	}
	return page, nil
}

func (s *Service) Unmatch(ctx context.Context, me, matchID string) error {
	other, err := s.repo.Delete(ctx, me, matchID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.events.Publish(other, realtime.Event{Type: "unmatched", Payload: map[string]string{"matchId": matchID}})
	return nil
}

func encodeCursor(c Cursor) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d|%s", c.SortAt.UnixMicro(), c.ID)))
}

func decodeCursor(raw string) (*Cursor, error) {
	if raw == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrBadCursor
	}
	var micro int64
	var id string
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return nil, ErrBadCursor
	}
	if _, err := fmt.Sscanf(parts[0], "%d", &micro); err != nil {
		return nil, ErrBadCursor
	}
	id = parts[1]
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrBadCursor
	}
	return &Cursor{SortAt: time.UnixMicro(micro).UTC(), ID: id}, nil
}
