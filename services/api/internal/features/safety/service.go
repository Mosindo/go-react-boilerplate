package safety

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var ErrInvalidReport = errors.New("safety: invalid report")

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	dm, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if dm != nil {
		for _, id := range dm.UserIDs {
			s.pub.Publish(id, realtime.Event{Type: "match.removed", Data: map[string]string{
				"matchId": dm.MatchID, "conversationId": dm.ConversationID,
			}})
		}
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string) ([]BlockItem, error) {
	return s.repo.ListBlocks(ctx, blockerID)
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) (string, error) {
	if !req.Reason.Valid() {
		return "", ErrInvalidReport
	}
	details := strings.Map(func(r rune) rune {
		if r == '\n' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, req.Details)
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > maxDetailsRunes {
		return "", ErrInvalidReport
	}
	return s.repo.CreateReport(ctx, reporterID, req.UserID, req.Reason, details)
}
