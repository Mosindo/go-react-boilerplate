package matches

import (
	"context"
	"errors"
	"strings"
	"time"

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

func (s *Service) Swipe(ctx context.Context, me string, req SwipeRequest) (SwipeResult, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "like" && action != "pass" {
		return SwipeResult{}, perr.Invalid("invalid_action", "action must be \"like\" or \"pass\"")
	}
	target := strings.TrimSpace(req.TargetID)
	if !isUUID(target) {
		return SwipeResult{}, perr.Invalid("invalid_target", "targetId is not valid")
	}
	if target == me {
		return SwipeResult{}, perr.Invalid("invalid_target", "you cannot swipe yourself")
	}

	out, err := s.repo.Swipe(ctx, me, target, action)
	if err != nil {
		switch {
		case errors.Is(err, ErrTargetUnavailable):
			return SwipeResult{}, perr.NotFound("profile_not_found", "profile not found")
		case errors.Is(err, ErrNoProfile):
			return SwipeResult{}, perr.Conflict("profile_incomplete", "create your profile before swiping")
		}
		return SwipeResult{}, err
	}

	result := SwipeResult{Action: out.Action}
	if out.Match != nil {
		m := s.toMatch(*out.Match)
		result.Matched, result.Match = true, &m
		if out.NewMatch {
			// The other member sees the caller as their new match.
			mine := Match{ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt,
				User: UserSummary{ID: me, FirstName: out.MyName, ThumbURL: s.signer.ThumbURL(out.MyPhotoID)}}
			s.publisher.Publish(me, realtime.Event{Type: "match.new", Data: m})
			s.publisher.Publish(target, realtime.Event{Type: "match.new", Data: mine})
			s.publisher.Publish(target, realtime.Event{Type: "notification.new", Data: map[string]string{"type": "match"}})
		}
	}
	return result, nil
}

func (s *Service) ResetPasses(ctx context.Context, me string) error {
	_, err := s.repo.ResetPasses(ctx, me)
	return err
}

func (s *Service) List(ctx context.Context, me string, before string, limit int) (MatchPage, error) {
	var cursor *time.Time
	if before != "" {
		t, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			return MatchPage{}, perr.Invalid("invalid_cursor", "before must be an RFC 3339 timestamp")
		}
		cursor = &t
	}
	rows, err := s.repo.List(ctx, me, cursor, limit+1)
	if err != nil {
		return MatchPage{}, err
	}
	page := MatchPage{Items: []Match{}}
	if len(rows) > limit {
		rows = rows[:limit]
		next := rows[len(rows)-1].CreatedAt.Format(time.RFC3339Nano)
		page.NextBefore = &next
	}
	for _, row := range rows {
		page.Items = append(page.Items, s.toMatch(row))
	}
	return page, nil
}

// Unmatch dissolves the match and its conversation for both members.
func (s *Service) Unmatch(ctx context.Context, me, matchID string) error {
	if !isUUID(matchID) {
		return perr.NotFound("match_not_found", "match not found")
	}
	other, conv, err := s.repo.Unmatch(ctx, me, matchID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return perr.NotFound("match_not_found", "match not found")
		}
		return err
	}
	ev := realtime.Event{Type: "conversation.removed", Data: map[string]string{"conversationId": conv}}
	s.publisher.Publish(other, ev)
	s.publisher.Publish(me, ev)
	return nil
}

func (s *Service) toMatch(m storedMatch) Match {
	return Match{
		ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt,
		User: UserSummary{ID: m.OtherID, FirstName: m.OtherName, ThumbURL: s.signer.ThumbURL(m.OtherPhotoID)},
	}
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
