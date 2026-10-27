package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
)

var (
	ErrInvalidUser    = errors.New("invalid user")
	ErrInvalidReason  = errors.New("invalid report reason")
	ErrDetailsTooLong = errors.New("details are too long")
	ErrUserNotFound   = errors.New("user not found")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func validTarget(self, target string) (string, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	if !httpx.IsUUID(target) || target == self {
		return "", ErrInvalidUser
	}
	return target, nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	target, err := validTarget(userID, targetID)
	if err != nil {
		return err
	}
	if err := s.repo.Block(ctx, userID, target); err != nil {
		if errors.Is(err, ErrRepoUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	target, err := validTarget(userID, targetID)
	if err != nil {
		return err
	}
	return s.repo.Unblock(ctx, userID, target)
}

func (s *Service) Blocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, userID)
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) (string, error) {
	target, err := validTarget(userID, req.UserID)
	if err != nil {
		return "", err
	}
	reason := strings.ToLower(strings.TrimSpace(req.Reason))
	if _, ok := validReasons[reason]; !ok {
		return "", ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > maxDetailsRunes {
		return "", ErrDetailsTooLong
	}
	id, err := s.repo.Report(ctx, userID, target, reason, details)
	if err != nil {
		if errors.Is(err, ErrRepoUserNotFound) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	if req.Block {
		if err := s.repo.Block(ctx, userID, target); err != nil {
			return "", err
		}
	}
	return id, nil
}
