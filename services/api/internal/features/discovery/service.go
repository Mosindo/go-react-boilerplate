package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
)

var (
	errIncomplete = platformerrors.Forbidden("profile_incomplete", "complete your profile (with at least one photo) first")
	errIneligible = platformerrors.NotFound("not_found", "profile not found")
)

type ProfileReader interface {
	PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type MatchHandler interface {
	HandleMatch(ctx context.Context, matchID, swiperID, otherID string) (chat.Conversation, error)
	Conversation(ctx context.Context, userID, matchID string) (chat.Conversation, error)
}

type Service struct {
	repo     Repository
	profiles ProfileReader
	matches  MatchHandler
}

func NewService(repo Repository, profiles ProfileReader, matches MatchHandler) *Service {
	return &Service{repo: repo, profiles: profiles, matches: matches}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) (DiscoverResponse, error) {
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return DiscoverResponse{}, err
	}
	if len(ids) == 0 {
		// Distinguish "nothing left" from "profile incomplete" so clients can route correctly.
		if err := s.requireComplete(ctx, userID); err != nil {
			return DiscoverResponse{}, err
		}
		return DiscoverResponse{Profiles: []profiles.PublicProfile{}}, nil
	}
	list, err := s.profiles.PublicByIDs(ctx, userID, ids)
	if err != nil {
		return DiscoverResponse{}, err
	}
	return DiscoverResponse{Profiles: list}, nil
}

func (s *Service) requireComplete(ctx context.Context, userID string) error {
	complete, err := s.repo.IsComplete(ctx, userID)
	if err != nil {
		return err
	}
	if !complete {
		return errIncomplete
	}
	return nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	if req.TargetUserID == userID {
		return SwipeResult{}, platformerrors.BadRequest("invalid_target", "you cannot swipe on yourself")
	}
	out, err := s.repo.Swipe(ctx, userID, req.TargetUserID, req.Action)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepoIncomplete):
			return SwipeResult{}, errIncomplete
		case errors.Is(err, ErrRepoIneligible):
			return SwipeResult{}, errIneligible
		}
		return SwipeResult{}, err
	}
	res := SwipeResult{Action: out.Action, Matched: out.MatchID != ""}
	if !res.Matched {
		return res, nil
	}
	var conv chat.Conversation
	if out.Created {
		conv, err = s.matches.HandleMatch(ctx, out.MatchID, userID, req.TargetUserID)
	} else {
		conv, err = s.matches.Conversation(ctx, userID, out.MatchID)
	}
	if err != nil {
		return SwipeResult{}, err
	}
	res.Conversation = &conv
	return res, nil
}

func (s *Service) ResetPasses(ctx context.Context, userID string) error {
	return s.repo.ResetPasses(ctx, userID)
}
