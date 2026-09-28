package safety

import (
	"context"
	"errors"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/validate"
)

const maxBlockList = 500

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func target(userID, raw string) (string, error) {
	id, ok := httpx.NormalizeUUID(raw)
	if !ok {
		return "", httpx.BadRequest("userId must be a user id")
	}
	if id == userID {
		return "", httpx.BadRequest("you cannot target yourself")
	}
	return id, nil
}

func (s *Service) Block(ctx context.Context, userID, raw string) error {
	id, err := target(userID, raw)
	if err != nil {
		return err
	}
	if err := s.repo.Block(ctx, userID, id); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return httpx.NotFound("user not found")
		}
		return err
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, raw string) error {
	id, ok := httpx.NormalizeUUID(raw)
	if !ok {
		return httpx.NotFound("user not found")
	}
	return s.repo.Unblock(ctx, userID, id)
}

func (s *Service) List(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, userID, maxBlockList)
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) (string, error) {
	id, err := target(userID, req.UserID)
	if err != nil {
		return "", err
	}
	if !ValidReason(req.Reason) {
		return "", httpx.BadRequest("reason must be one of spam, fake_profile, harassment, inappropriate_content, underage, other")
	}
	details, err := validate.Text(req.Details, 0, MaxDetails, true)
	if err != nil {
		return "", httpx.BadRequest("details must be at most 1000 characters")
	}
	reportID, err := s.repo.CreateReport(ctx, userID, id, req.Reason, details)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", httpx.NotFound("user not found")
		}
		return "", err
	}
	return reportID, nil
}
