package matching

import (
	"context"
	"log"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

const (
	discoveryPoolSize = 200
	defaultPageSize   = 10
)

// Notifier persists an in-app notification and pushes it in real time.
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string)
}

type Service struct {
	repo     Repository
	profiles *profiles.Service
	ranker   Ranker
	notifier Notifier
	events   realtime.Publisher
	now      func() time.Time
}

func NewService(repo Repository, profileService *profiles.Service, ranker Ranker, notifier Notifier, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profileService, ranker: ranker, notifier: notifier, events: events, now: time.Now}
}

// Discover returns the next ranked profiles for userID.
func (s *Service) Discover(ctx context.Context, userID string, limit int) (DiscoveryResponse, error) {
	if err := s.repo.TouchActivity(ctx, userID); err != nil {
		log.Printf(`{"event":"touch_activity_failed","error":%q}`, err.Error())
	}
	own, err := s.profiles.GetOwn(ctx, userID)
	if err != nil {
		return DiscoveryResponse{}, err
	}
	switch {
	case !own.IsComplete:
		return DiscoveryResponse{Profiles: []profiles.Card{}, Status: "incomplete_profile", MissingFields: own.MissingFields}, nil
	case own.ModerationStatus != "active":
		return DiscoveryResponse{Profiles: []profiles.Card{}, Status: "under_review"}, nil
	case !own.Discoverable:
		// Paused members can still browse; they simply are not shown to others.
	}

	pool, maxDistance, err := s.repo.Candidates(ctx, userID, discoveryPoolSize)
	if err != nil {
		return DiscoveryResponse{}, err
	}
	ranked := s.ranker.Rank(s.now(), maxDistance, pool)
	if limit <= 0 || limit > 20 {
		limit = defaultPageSize
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, 0, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return DiscoveryResponse{}, err
	}
	status := "ok"
	if !own.Discoverable {
		status = "paused"
	}
	return DiscoveryResponse{Profiles: cards, Status: status}, nil
}

func (s *Service) Swipe(ctx context.Context, swiperID string, req SwipeRequest) (SwipeResponse, error) {
	if swiperID == req.TargetUserID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	res, err := s.repo.Swipe(ctx, swiperID, req.TargetUserID, req.Action)
	if err != nil {
		return SwipeResponse{}, err
	}
	out := SwipeResponse{Action: res.Action}
	if res.Match == nil {
		if res.NewLike {
			s.notifier.Notify(ctx, req.TargetUserID, "like.received", "Quelqu'un vous a liké", "Ouvrez l'onglet J'aime pour découvrir qui.", nil)
			s.events.Publish(req.TargetUserID, "like.received", map[string]string{})
		}
		return out, nil
	}

	mine, err := s.matchView(ctx, swiperID, *res.Match)
	if err != nil {
		return SwipeResponse{}, err
	}
	out.Matched = true
	out.Match = &mine

	if res.NewMatch {
		// "theirs" is the same match seen by the target: its User is the swiper.
		theirs, err := s.matchView(ctx, req.TargetUserID, storedMatch{
			ID: res.Match.ID, OtherUserID: swiperID, ConversationID: res.Match.ConversationID, CreatedAt: res.Match.CreatedAt,
		})
		swiperName := "un membre"
		if err == nil {
			s.events.Publish(req.TargetUserID, "match.created", theirs)
			if theirs.User.FirstName != "" {
				swiperName = theirs.User.FirstName
			}
		}
		s.events.Publish(swiperID, "match.created", mine)
		s.notifier.Notify(ctx, req.TargetUserID, "match.created", "Nouveau match !",
			"Vous et "+swiperName+" vous plaisez mutuellement.", map[string]string{"matchId": res.Match.ID, "userId": swiperID})
		s.notifier.Notify(ctx, swiperID, "match.created", "Nouveau match !",
			"Vous et "+mine.User.FirstName+" vous plaisez mutuellement.", map[string]string{"matchId": res.Match.ID, "userId": req.TargetUserID})
	}
	return out, nil
}

func (s *Service) matchView(ctx context.Context, viewerID string, m storedMatch) (MatchView, error) {
	cards, err := s.profiles.Cards(ctx, viewerID, []string{m.OtherUserID})
	if err != nil {
		return MatchView{}, err
	}
	view := MatchView{ID: m.ID, MatchedAt: m.CreatedAt}
	if m.ConversationID != nil {
		view.ConversationID = *m.ConversationID
	}
	if len(cards) > 0 {
		view.User = cards[0]
	} else {
		view.User = profiles.Card{UserID: m.OtherUserID, Interests: []profiles.Interest{}, Photos: []profiles.Photo{}}
	}
	return view, nil
}

func (s *Service) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]profiles.Card, error) {
	ids, err := s.repo.LikesReceived(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.profiles.Cards(ctx, userID, ids)
}

func (s *Service) Matches(ctx context.Context, userID string, limit, offset int) ([]MatchView, error) {
	stored, err := s.repo.Matches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(stored))
	for _, m := range stored {
		ids = append(ids, m.OtherUserID)
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]MatchView, 0, len(stored))
	for _, m := range stored {
		card, ok := byID[m.OtherUserID]
		if !ok {
			continue
		}
		v := MatchView{ID: m.ID, MatchedAt: m.CreatedAt, User: card}
		if m.ConversationID != nil {
			v.ConversationID = *m.ConversationID
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, otherID string) error {
	if err := s.repo.Unmatch(ctx, userID, otherID); err != nil {
		return err
	}
	s.events.Publish(otherID, "match.removed", map[string]string{"userId": userID})
	s.events.Publish(userID, "match.removed", map[string]string{"userId": otherID})
	return nil
}

func (s *Service) ResetPasses(ctx context.Context, userID string) (int, error) {
	return s.repo.ResetPasses(ctx, userID)
}
