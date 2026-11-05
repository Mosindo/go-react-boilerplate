package discovery

import "context"

const (
	DefaultFeedSize = 10
	MaxFeedSize     = 20
)

type Service struct {
	repo   Repository
	events Events
}

func NewService(repo Repository, events Events) *Service {
	return &Service{repo: repo, events: events}
}

// Feed returns the next unseen, mutually-eligible profiles. Already-swiped,
// blocked and non-matching profiles are excluded by the query itself, so no
// cursor is needed: acting on a card removes it from the next page.
func (s *Service) Feed(ctx context.Context, userID string, limit int) ([]Candidate, error) {
	_ = s.repo.Touch(ctx, userID)
	return s.repo.Candidates(ctx, userID, limit)
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (Candidate, error) {
	if viewerID == targetID {
		return Candidate{}, ErrNotFound
	}
	return s.repo.Visible(ctx, viewerID, targetID)
}

func (s *Service) Swipe(ctx context.Context, viewerID string, req SwipeRequest) (SwipeResult, error) {
	if req.UserID == viewerID {
		return SwipeResult{}, ErrNotEligible
	}
	res, info, err := s.repo.Swipe(ctx, viewerID, req.UserID, req.Action)
	if err != nil {
		return SwipeResult{}, err
	}
	if info != nil && s.events != nil {
		s.events.MatchCreated(*info)
	}
	return res, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	convID, other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	if s.events != nil {
		s.events.MatchEnded(matchID, convID, userID, other)
	}
	return nil
}
