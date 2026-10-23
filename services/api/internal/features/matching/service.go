package matching

import (
	"context"
	"errors"
	"log"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"example.com/api/internal/platform/realtime"
)

// Notifier creates user-facing notifications (implemented by the notifications feature).
type Notifier interface {
	NotifyMatch(ctx context.Context, userID, matchID, otherUserID, otherName string) error
}

type Service struct {
	repo     Repository
	notifier Notifier
	pub      realtime.Publisher
}

func NewService(repo Repository, notifier Notifier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, notifier: notifier, pub: pub}
}

func (s *Service) Swipe(ctx context.Context, swiperID string, req SwipeRequest) (SwipeResponse, error) {
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResponse{}, platformerrors.Validation(map[string]string{"action": "must be like or pass"})
	}
	if req.TargetUserID == swiperID {
		return SwipeResponse{}, platformerrors.BadRequest("invalid_target", "you cannot swipe yourself")
	}
	out, err := s.repo.Swipe(ctx, swiperID, req.TargetUserID, req.Action)
	if err != nil {
		if errors.Is(err, ErrRepositoryTargetUnavailable) {
			return SwipeResponse{}, platformerrors.NotFound("profile_not_found", "profile not found")
		}
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{Action: out.Action, AlreadyProcessed: out.AlreadyProcessed}
	if out.MatchID == "" {
		return resp, nil
	}
	mine, err := s.repo.Summary(ctx, swiperID, out.MatchID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return resp, nil
		}
		return SwipeResponse{}, err
	}
	resp.Matched = true
	resp.Match = &mine

	if out.NewMatch {
		s.announceMatch(ctx, swiperID, req.TargetUserID, mine)
	}
	return resp, nil
}

// announceMatch is best-effort: the match is already committed.
func (s *Service) announceMatch(ctx context.Context, swiperID, targetID string, mine MatchSummary) {
	theirs, err := s.repo.Summary(ctx, targetID, mine.ID)
	if err != nil {
		log.Printf(`{"event":"match_announce_failed","error":%q}`, err.Error())
		return
	}
	s.pub.Publish(swiperID, "match.new", mine)
	s.pub.Publish(targetID, "match.new", theirs)
	// theirs.User is the swiper from the target's point of view.
	if err := s.notifier.NotifyMatch(ctx, targetID, mine.ID, swiperID, theirs.User.FirstName); err != nil {
		log.Printf(`{"event":"match_notification_failed","error":%q}`, err.Error())
	}
}

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string) (MatchList, error) {
	var before *time.Time
	var beforeID string
	if cursor != "" {
		t, id, ok := pagination.Decode(cursor)
		if !ok {
			return MatchList{}, platformerrors.BadRequest("invalid_cursor", "invalid cursor")
		}
		before, beforeID = &t, id
	}
	items, err := s.repo.List(ctx, userID, limit+1, before, beforeID)
	if err != nil {
		return MatchList{}, err
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		c := pagination.Encode(last.LastActivityAt, last.ID)
		next = &c
	}
	return MatchList{Matches: items, NextCursor: next}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return platformerrors.NotFound("match_not_found", "match not found")
		}
		return err
	}
	payload := map[string]string{"matchId": matchID}
	s.pub.Publish(userID, "match.removed", payload)
	s.pub.Publish(other, "match.removed", payload)
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, matchID string) error {
	if err := s.repo.Hide(ctx, userID, matchID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return platformerrors.NotFound("match_not_found", "match not found")
		}
		return err
	}
	return nil
}

func notFoundProfile() error {
	return platformerrors.NotFound("profile_not_found", "profile not found")
}
