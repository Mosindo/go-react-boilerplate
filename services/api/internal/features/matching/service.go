package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile to discover people")
	ErrProfilePaused     = errors.New("your profile is paused")
	ErrProfileInReview   = errors.New("your profile is under review")
	ErrSelfSwipe         = errors.New("you cannot swipe on yourself")
)

// CardLoader hydrates public profiles (implemented by profiles.Service).
type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

// Notifier stores in-app notifications (implemented by notifications.Service).
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string, dedupeKey string)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	notifier  Notifier
	publisher realtime.Publisher
	ranker    Ranker
	now       func() time.Time
}

func NewService(repo Repository, cards CardLoader, notifier Notifier, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.Nop{}
	}
	return &Service{
		repo:      repo,
		cards:     cards,
		notifier:  notifier,
		publisher: publisher,
		ranker:    DefaultRanker(),
		now:       time.Now,
	}
}

// Discover returns the next ranked profiles. Already-answered profiles are
// excluded server-side, so clients simply call it again for more.
func (s *Service) Discover(ctx context.Context, userID, organizationID string, limit int) ([]profiles.PublicProfile, error) {
	if limit <= 0 || limit > maxDiscoveryLimit {
		limit = defaultDiscoveryLimit
	}
	if err := s.ensureCanBrowse(ctx, userID); err != nil {
		return nil, err
	}
	_ = s.repo.TouchActivity(ctx, userID)

	pool, err := s.repo.CandidatePool(ctx, userID, organizationID, candidatePoolSize)
	if err != nil {
		return nil, err
	}
	ranked := s.ranker.Rank(s.now(), pool)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, 0, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
	}
	return s.cards.Cards(ctx, userID, ids)
}

func (s *Service) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResult, error) {
	if swiperID == targetID {
		return SwipeResult{}, ErrSelfSwipe
	}
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, errors.New("invalid action")
	}
	if err := s.ensureCanBrowse(ctx, swiperID); err != nil {
		return SwipeResult{}, err
	}
	canSwipe, err := s.repo.CanSwipe(ctx, swiperID, targetID)
	if err != nil {
		return SwipeResult{}, err
	}
	if !canSwipe {
		return SwipeResult{}, ErrTargetUnavailable
	}

	outcome, err := s.repo.Swipe(ctx, swiperID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	result := SwipeResult{Action: action, Matched: outcome.Matched}
	if !outcome.Matched {
		return result, nil
	}

	cards, err := s.cards.Cards(ctx, swiperID, []string{targetID})
	if err != nil {
		return SwipeResult{}, err
	}
	if len(cards) == 1 {
		result.Match = &MatchInfo{ID: outcome.MatchID, CreatedAt: outcome.CreatedAt, User: cards[0]}
	}
	if outcome.New {
		s.announceMatch(ctx, outcome, swiperID, targetID)
	}
	return result, nil
}

// ensureCanBrowse: incomplete, paused or moderation-hidden members can
// neither browse nor create new matches (a hidden profile must not keep
// collecting contacts while it is reviewed).
func (s *Service) ensureCanBrowse(ctx context.Context, userID string) error {
	st, err := s.repo.ProfileStatus(ctx, userID)
	if err != nil {
		return err
	}
	switch {
	case st.Hidden:
		return ErrProfileInReview
	case !st.Complete:
		return ErrProfileIncomplete
	case st.Paused:
		return ErrProfilePaused
	}
	return nil
}

// announceMatch notifies both members; each receives the other's card.
func (s *Service) announceMatch(ctx context.Context, outcome swipeOutcome, a, b string) {
	for _, pair := range [][2]string{{a, b}, {b, a}} {
		viewer, other := pair[0], pair[1]
		cards, err := s.cards.Cards(ctx, viewer, []string{other})
		if err != nil || len(cards) == 0 {
			continue
		}
		card := cards[0]
		s.publisher.Publish(viewer, "match.created", MatchInfo{ID: outcome.MatchID, CreatedAt: outcome.CreatedAt, User: card})
		if s.notifier != nil {
			s.notifier.Notify(ctx, viewer, notifications.TypeMatch,
				"Nouveau match !",
				"Vous et "+card.FirstName+" vous plaisez mutuellement. Lancez la conversation.",
				map[string]string{"peerUserId": other, "matchId": outcome.MatchID}, "")
		}
	}
}

func (s *Service) Unmatch(ctx context.Context, userID, otherUserID string) error {
	if err := s.repo.Unmatch(ctx, userID, otherUserID); err != nil {
		return err
	}
	s.publisher.Publish(otherUserID, "match.removed", map[string]string{"userId": userID})
	s.publisher.Publish(userID, "match.removed", map[string]string{"userId": otherUserID})
	return nil
}
