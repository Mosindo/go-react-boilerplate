package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo      Repository
	signer    *storage.Signer
	publisher realtime.Publisher
}

func NewService(repo Repository, signer *storage.Signer, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, signer: signer, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, me, target string) error {
	if err := validateTarget(me, target); err != nil {
		return err
	}
	convs, err := s.repo.Block(ctx, me, target)
	if err != nil {
		return mapTarget(err)
	}
	for _, id := range convs {
		ev := realtime.Event{Type: "conversation.removed", Data: map[string]string{"conversationId": id}}
		s.publisher.Publish(me, ev)
		s.publisher.Publish(target, ev)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, me, target string) error {
	if !isUUID(target) {
		return perr.NotFound("user_not_found", "user not found")
	}
	return s.repo.Unblock(ctx, me, target)
}

func (s *Service) List(ctx context.Context, me string) ([]BlockedUser, error) {
	rows, err := s.repo.ListBlocks(ctx, me)
	if err != nil {
		return nil, err
	}
	out := make([]BlockedUser, 0, len(rows))
	for _, b := range rows {
		out = append(out, BlockedUser{
			User:      storage.UserSummary{ID: b.ID, FirstName: b.Name, ThumbURL: s.signer.ThumbURL(b.PhotoID)},
			BlockedAt: b.BlockedAt,
		})
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, me string, req ReportRequest) (Report, error) {
	if err := validateTarget(me, req.UserID); err != nil {
		return Report{}, err
	}
	reason := strings.TrimSpace(req.Reason)
	valid := false
	for _, r := range ReportReasons {
		valid = valid || r == reason
	}
	if !valid {
		return Report{}, perr.Invalid("invalid_reason", "report reason is not valid")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return Report{}, perr.Invalid("invalid_details", "details must be at most 1000 characters")
	}
	rep, err := s.repo.CreateReport(ctx, me, req.UserID, reason, details)
	if err != nil {
		return Report{}, mapTarget(err)
	}
	if req.AlsoBlock {
		if err := s.Block(ctx, me, req.UserID); err != nil {
			return Report{}, err
		}
	}
	return rep, nil
}

func validateTarget(me, target string) error {
	if !isUUID(target) {
		return perr.NotFound("user_not_found", "user not found")
	}
	if target == me {
		return perr.Invalid("invalid_target", "you cannot target yourself")
	}
	return nil
}

func mapTarget(err error) error {
	if errors.Is(err, ErrTargetNotFound) {
		return perr.NotFound("user_not_found", "user not found")
	}
	return err
}

func isUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, r := range v {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'):
			return false
		}
	}
	return true
}
