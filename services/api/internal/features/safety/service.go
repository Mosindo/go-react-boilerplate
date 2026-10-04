package safety

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
	"github.com/google/uuid"
)

type Service struct {
	repo Repository
	rt   realtime.Publisher
}

func NewService(repo Repository, rt realtime.Publisher) *Service { return &Service{repo: repo, rt: rt} }

func validTarget(self, target string) error {
	if _, err := uuid.Parse(target); err != nil {
		return platformerrors.FieldInvalid("userId", "Invalid user id")
	}
	if self == target {
		return platformerrors.FieldInvalid("userId", "You cannot target yourself")
	}
	return nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if err := validTarget(userID, targetID); err != nil {
		return err
	}
	removed, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return mapErr(err)
	}
	s.publishRemoved(removed)
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	if _, err := uuid.Parse(targetID); err != nil {
		return platformerrors.FieldInvalid("userId", "Invalid user id")
	}
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) List(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, userID)
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) (string, error) {
	if err := validTarget(userID, req.UserID); err != nil {
		return "", err
	}
	if _, ok := Reasons[req.Reason]; !ok {
		return "", platformerrors.FieldInvalid("reason", "Unknown reason")
	}
	details := cleanDetails(req.Details)
	if utf8.RuneCountInString(details) > maxDetailsRunes {
		return "", platformerrors.FieldInvalid("details", "Use at most 1000 characters")
	}
	id, removed, err := s.repo.CreateReport(ctx, userID, req.UserID, req.Reason, details, req.Block)
	if err != nil {
		return "", mapErr(err)
	}
	s.publishRemoved(removed)
	return id, nil
}

func (s *Service) publishRemoved(rm *RemovedMatch) {
	if rm == nil {
		return
	}
	for _, uid := range rm.UserIDs {
		s.rt.Publish(uid, "match.removed", map[string]string{"matchId": rm.MatchID, "conversationId": rm.ConversationID})
	}
}

func mapErr(err error) error {
	if errors.Is(err, ErrUserNotFound) {
		return platformerrors.NotFound("User not found")
	}
	return err
}

func cleanDetails(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
