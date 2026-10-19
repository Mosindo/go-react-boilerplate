package safety

import (
	"context"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrSelfAction
	}
	removed, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if removed != nil {
		s.pub.Publish(targetID, realtime.Event{Type: "match.removed", Data: map[string]string{
			"matchId": removed.MatchID, "conversationId": removed.ConversationID,
		}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) ListBlocks(ctx context.Context, userID string) (BlocksResponse, error) {
	blocks, err := s.repo.ListBlocks(ctx, userID)
	return BlocksResponse{Blocks: blocks}, err
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	if userID == req.UserID {
		return ErrSelfAction
	}
	if _, ok := validReasons[req.Reason]; !ok {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > MaxDetailsRunes || !utf8.ValidString(details) {
		return ErrInvalidInput
	}
	return s.repo.Report(ctx, userID, req.UserID, req.Reason, details)
}
