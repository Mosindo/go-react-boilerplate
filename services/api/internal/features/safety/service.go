package safety

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrSelf
	}
	closed, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	s.announce(userID, targetID, closed)
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) Blocks(ctx context.Context, userID string) ([]BlockedUser, error) {
	stored, err := s.repo.ListBlocks(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]BlockedUser, 0, len(stored))
	for _, b := range stored {
		name := "Profil supprimé"
		if b.FirstName != nil {
			name = *b.FirstName
		}
		out = append(out, BlockedUser{UserID: b.UserID, FirstName: name, BlockedAt: b.CreatedAt})
	}
	return out, nil
}

// Report always blocks the reported user as well, so the reporter is protected immediately.
func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	target := strings.ToLower(req.UserID)
	if userID == target {
		return ErrSelf
	}
	reason := strings.ToLower(strings.TrimSpace(req.Reason))
	if !slices.Contains(Reasons, reason) {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return ErrInvalidDetail
	}
	closed, err := s.repo.Report(ctx, userID, target, reason, details)
	if err != nil {
		return err
	}
	s.announce(userID, target, closed)
	return nil
}

func (s *Service) announce(userID, targetID string, closed *Closed) {
	if closed == nil {
		return
	}
	event := realtime.Event{Type: "conversation.closed", Payload: map[string]string{
		"conversationId": closed.ConversationID,
		"matchId":        closed.MatchID,
	}}
	s.publisher.Publish(userID, event)
	s.publisher.Publish(targetID, event)
}
