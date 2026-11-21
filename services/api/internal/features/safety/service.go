package safety

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelfAction    = errors.New("you cannot block or report yourself")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrDetailsLength = errors.New("details must be at most 1000 characters")
)

type Service struct {
	repo      Repository
	profiles  *profiles.Service
	publisher realtime.Publisher
}

func NewService(repo Repository, profilesService *profiles.Service, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.Nop{}
	}
	return &Service{repo: repo, profiles: profilesService, publisher: publisher}
}

// NormalizeReport validates a report payload.
func NormalizeReport(reason, details string) (string, string, error) {
	r := strings.ToLower(strings.TrimSpace(reason))
	valid := false
	for _, allowed := range ReportReasons {
		if r == allowed {
			valid = true
			break
		}
	}
	if !valid {
		return "", "", ErrInvalidReason
	}
	cleaned := strings.Map(func(ch rune) rune {
		if ch != '\n' && unicode.IsControl(ch) {
			return -1
		}
		return ch
	}, strings.TrimSpace(details))
	if utf8.RuneCountInString(cleaned) > 1000 {
		return "", "", ErrDetailsLength
	}
	return r, cleaned, nil
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrSelfAction
	}
	hadMatch, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	s.notifyRemoval(userID, targetID, hadMatch)
	return nil
}

// notifyRemoval tells clients to drop the conversation. The blocked member
// only sees the match disappear, never that they were blocked.
func (s *Service) notifyRemoval(userID, targetID string, hadMatch bool) {
	s.publisher.Publish(userID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": targetID}})
	if hadMatch {
		s.publisher.Publish(targetID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": userID}})
	}
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) ListBlocks(ctx context.Context, userID string, limit, offset int) ([]BlockedUserResponse, error) {
	rows, err := s.repo.ListBlocks(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserID)
	}
	summaries, err := s.profiles.Summaries(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]BlockedUserResponse, 0, len(rows))
	for _, r := range rows {
		summary, ok := summaries[r.UserID]
		if !ok {
			summary = profiles.DeletedSummary(r.UserID)
		}
		out = append(out, BlockedUserResponse{User: summary, BlockedAt: r.CreatedAt})
	}
	return out, nil
}

// Report always blocks the reported member as well: the reporter should never
// have to see that person again.
func (s *Service) Report(ctx context.Context, userID, targetID, reason, details string) (ReportResponse, error) {
	if userID == targetID {
		return ReportResponse{}, ErrSelfAction
	}
	normalizedReason, normalizedDetails, err := NormalizeReport(reason, details)
	if err != nil {
		return ReportResponse{}, err
	}
	reportID, err := s.repo.CreateReport(ctx, userID, targetID, normalizedReason, normalizedDetails, AutoSuspendThreshold)
	if err != nil {
		return ReportResponse{}, err
	}
	s.notifyRemoval(userID, targetID, true)
	return ReportResponse{ID: reportID, Blocked: true}, nil
}
