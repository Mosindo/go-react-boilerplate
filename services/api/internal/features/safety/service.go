package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
)

var (
	errUserNotFound = platformerrors.NotFound("not_found", "user not found")
	errSelf         = platformerrors.BadRequest("invalid_target", "you cannot do this to yourself")
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return errSelf
	}
	if err := s.repo.Block(ctx, userID, targetID); err != nil {
		if errors.Is(err, ErrRepoUserNotFound) {
			return errUserNotFound
		}
		return err
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) List(ctx context.Context, userID string) ([]Block, error) {
	list, err := s.repo.List(ctx, userID)
	if list == nil {
		list = []Block{}
	}
	return list, err
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) (string, error) {
	if userID == req.UserID {
		return "", errSelf
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return "", platformerrors.BadRequest("details_too_long", "details must be at most 1000 characters")
	}
	id, err := s.repo.Report(ctx, userID, req.UserID, req.ConversationID, req.Reason, details)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepoUserNotFound):
			return "", errUserNotFound
		case errors.Is(err, ErrRepoBadMatch):
			return "", platformerrors.BadRequest("invalid_conversation", "the conversation does not involve both users")
		}
		return "", err
	}
	return id, nil
}
