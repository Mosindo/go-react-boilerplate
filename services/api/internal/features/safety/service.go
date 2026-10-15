package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/users"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrUserNotFound  = platformerrors.NotFound("user_not_found", "user not found")
	ErrSelfAction    = platformerrors.BadRequest("invalid_target", "you cannot do this to yourself")
	ErrInvalidReason = platformerrors.BadRequest("invalid_reason", "the report reason is not valid")
	ErrInvalidDetail = platformerrors.BadRequest("invalid_details", "details are too long (1000 characters at most)")
)

type Publisher interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo      Repository
	photos    users.PhotoSource
	publisher Publisher
}

func NewService(repo Repository, photos users.PhotoSource, publisher Publisher) *Service {
	return &Service{repo: repo, photos: photos, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	blockedID = strings.ToLower(blockedID)
	if blockerID == blockedID {
		return ErrSelfAction
	}
	conversationID, matched, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if matched {
		s.publisher.Publish(blockedID, realtime.Event{Type: "match.removed", Data: map[string]any{
			"userId":         blockerID,
			"conversationId": conversationID,
		}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, strings.ToLower(blockedID))
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	list, err := s.repo.ListBlocked(ctx, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, u := range list {
		ids = append(ids, u.UserID)
	}
	photos, err := s.photos.PhotosForUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if p := photos[list[i].UserID]; len(p) > 0 {
			thumb := p[0].ThumbURL
			list[i].ThumbURL = &thumb
		}
	}
	return list, nil
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	reportedID := strings.ToLower(req.UserID)
	if reporterID == reportedID {
		return ErrSelfAction
	}
	reason := strings.TrimSpace(req.Reason)
	if _, ok := validReasons[reason]; !ok {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > maxDetailsRunes {
		return ErrInvalidDetail
	}
	if err := s.repo.CreateReport(ctx, reporterID, reportedID, reason, details); err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}
