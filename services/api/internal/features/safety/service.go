package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrSelfAction    = errors.New("you cannot do this on yourself")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrDetailsLength = errors.New("details must be at most 1000 characters")
)

const maxDetailsRunes = 1000

type URLSigner interface {
	PhotoURL(photoID string) string
}

type Service struct {
	repo      Repository
	signer    URLSigner
	publisher realtime.Publisher
}

func NewService(repo Repository, signer URLSigner, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, signer: signer, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelfAction
	}
	hadMatch, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if hadMatch {
		s.announceRemoval(blockerID, blockedID)
	}
	return nil
}

// announceRemoval makes the conversation disappear on both sides. The blocked
// member sees the same thing as an unmatch: blocks are never disclosed.
func (s *Service) announceRemoval(a, b string) {
	s.publisher.Publish(a, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": b}})
	s.publisher.Publish(b, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": a}})
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	if err := s.repo.Unblock(ctx, blockerID, blockedID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	stored, err := s.repo.ListBlocks(ctx, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]BlockedUser, 0, len(stored))
	for _, b := range stored {
		item := BlockedUser{UserID: b.UserID, BlockedAt: b.BlockedAt, FirstName: "Membre"}
		if b.FirstName != nil {
			item.FirstName = *b.FirstName
		}
		if b.PhotoID != nil {
			url := s.signer.PhotoURL(*b.PhotoID)
			item.PhotoURL = &url
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if reporterID == reportedID {
		return ErrSelfAction
	}
	reason = strings.TrimSpace(reason)
	valid := false
	for _, r := range ReportReasons {
		if r == reason {
			valid = true
			break
		}
	}
	if !valid {
		return ErrInvalidReason
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > maxDetailsRunes {
		return ErrDetailsLength
	}
	if err := s.repo.Report(ctx, reporterID, reportedID, reason, details); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	// Reporting always blocks, so the reporter never sees this member again.
	s.announceRemoval(reporterID, reportedID)
	return nil
}
