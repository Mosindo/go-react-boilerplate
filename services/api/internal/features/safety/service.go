package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
)

var (
	ErrInvalidUser   = errors.New("invalid user")
	ErrInvalidReason = errors.New("unknown report reason")
	ErrDetailsLength = errors.New("details must be at most 1000 characters")
)

type ProfileSource interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileSource
}

func NewService(repo Repository, source ProfileSource) *Service {
	return &Service{repo: repo, profiles: source}
}

func validTarget(actorID, targetID string) error {
	if !httpx.IsUUID(targetID) || strings.EqualFold(actorID, targetID) {
		return ErrInvalidUser
	}
	return nil
}

func (s *Service) Block(ctx context.Context, actorID, targetID string) error {
	if err := validTarget(actorID, targetID); err != nil {
		return err
	}
	return s.repo.Block(ctx, actorID, strings.ToLower(targetID))
}

func (s *Service) Unblock(ctx context.Context, actorID, targetID string) error {
	if err := validTarget(actorID, targetID); err != nil {
		return err
	}
	return s.repo.Unblock(ctx, actorID, strings.ToLower(targetID))
}

func (s *Service) Report(ctx context.Context, actorID string, req ReportRequest) error {
	if err := validTarget(actorID, req.UserID); err != nil {
		return err
	}
	if _, ok := reportReasons[req.Reason]; !ok {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return ErrDetailsLength
	}
	return s.repo.Report(ctx, actorID, strings.ToLower(req.UserID), req.Reason, details)
}

func (s *Service) Blocked(ctx context.Context, actorID string) ([]BlockedUser, error) {
	stored, err := s.repo.Blocks(ctx, actorID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(stored))
	for i, b := range stored {
		ids[i] = b.BlockedID
	}
	cards, err := s.profiles.Cards(ctx, actorID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.PublicProfile, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	out := make([]BlockedUser, 0, len(stored))
	for _, b := range stored {
		if card, ok := byID[b.BlockedID]; ok {
			out = append(out, BlockedUser{BlockedAt: b.CreatedAt, Profile: card})
		}
	}
	return out, nil
}
