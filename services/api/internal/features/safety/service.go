package safety

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrValidation = errors.New("invalid request")
	ErrNotFound   = errors.New("user not found")
)

type Cards interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.Card, error)
}

type Service struct {
	repo   Repository
	cards  Cards
	events realtime.Publisher
}

func NewService(repo Repository, cards Cards, events realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, events: events}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrValidation
	}
	matchID, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrNotFound
		}
		return err
	}
	if matchID != "" {
		s.events.Publish(targetID, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) List(ctx context.Context, userID string) ([]BlockedUser, error) {
	stored, err := s.repo.ListBlocks(ctx, userID, 200)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(stored))
	for _, b := range stored {
		ids = append(ids, b.UserID)
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := map[string]profiles.Card{}
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]BlockedUser, 0, len(stored))
	for _, b := range stored {
		u := BlockedUser{UserID: b.UserID, BlockedAt: b.BlockedAt}
		if c, ok := byID[b.UserID]; ok {
			u.FirstName = c.FirstName
			if len(c.Photos) > 0 {
				u.ThumbURL = c.Photos[0].ThumbURL
			}
		}
		out = append(out, u)
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	if userID == req.UserID || !slices.Contains(Reasons, req.Reason) {
		return ErrValidation
	}
	details := strings.TrimSpace(req.Details)
	if !utf8.ValidString(details) || utf8.RuneCountInString(details) > 1000 {
		return ErrValidation
	}
	// Block first when asked: the report's message snapshot is taken afterwards
	// from chat_messages, so order matters. Take the snapshot before the block deletes the match.
	if err := s.repo.Report(ctx, userID, req.UserID, req.Reason, details); err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrNotFound
		}
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, userID, req.UserID)
	}
	return nil
}
