package safety

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelfAction    = errors.New("you cannot do this to yourself")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrDetailsLength = errors.New("details must be at most 1000 characters")
)

type ProfileReader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

type Service struct {
	repo     Repository
	profiles ProfileReader
	events   realtime.Publisher
}

func NewService(repo Repository, profiles ProfileReader, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, events: events}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelfAction
	}
	matchID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if matchID != "" {
		// Both clients drop the conversation. The blocked user is not told why.
		for _, id := range []string{blockerID, blockedID} {
			s.events.Publish(id, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
		}
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	limit = min(max(limit, 1), 100)
	rows, err := s.repo.ListBlocks(ctx, blockerID, limit, max(offset, 0))
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserID)
	}
	cards, err := s.profiles.Cards(ctx, blockerID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]BlockedUser, 0, len(rows))
	for _, r := range rows {
		card, ok := cards[r.UserID]
		if !ok {
			card = profiles.Card{ID: r.UserID, FirstName: "Utilisateur", Interests: []profiles.Interest{}, Photos: []profiles.Photo{}}
		}
		out = append(out, BlockedUser{User: card, BlockedAt: r.BlockedAt})
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	if reporterID == req.UserID {
		return ErrSelfAction
	}
	valid := false
	for _, r := range ReportReasons {
		if r == req.Reason {
			valid = true
		}
	}
	if !valid {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, req.Details))
	if utf8.RuneCountInString(details) > 1000 {
		return ErrDetailsLength
	}
	if err := s.repo.CreateReport(ctx, reporterID, req.UserID, req.Reason, details); err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, reporterID, req.UserID)
	}
	return nil
}
