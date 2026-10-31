package safety

import (
	"context"
	"errors"

	"example.com/api/internal/platform/validate"
	"github.com/google/uuid"
)

var (
	ErrInvalidTarget = errors.New("invalid user")
	ErrSelf          = errors.New("you cannot do this to yourself")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrInvalidText   = errors.New("details too long")
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func parseTarget(userID, raw string) (string, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", ErrInvalidTarget
	}
	if id.String() == userID {
		return "", ErrSelf
	}
	return id.String(), nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	target, err := parseTarget(userID, targetID)
	if err != nil {
		return err
	}
	return s.repo.Block(ctx, userID, target)
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	target, err := parseTarget(userID, targetID)
	if err != nil {
		return err
	}
	return s.repo.Unblock(ctx, userID, target)
}

func (s *Service) Blocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, userID, 200)
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	target, err := parseTarget(userID, req.UserID)
	if err != nil {
		return err
	}
	if !validReasons[req.Reason] {
		return ErrInvalidReason
	}
	details, ok := validate.CleanText(req.Details, 1000)
	if !ok {
		return ErrInvalidText
	}
	if err := s.repo.Report(ctx, userID, target, req.Reason, details); err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.repo.Block(ctx, userID, target)
	}
	return nil
}
