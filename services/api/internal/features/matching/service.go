package matching

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile (photo, details and location) to discover people")
	ErrInvalidAction     = errors.New("action must be like or pass")
	ErrSelfSwipe         = errors.New("you cannot swipe yourself")
)

type CardLoader interface {
	LoadCards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	notifier  Notifier
	publisher realtime.Publisher
	now       func() time.Time
}

func NewService(repo Repository, cards CardLoader, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, publisher: publisher, now: time.Now}
}

// BuildQuery turns a viewer into a resolved discovery filter.
func BuildQuery(v Viewer, today time.Time, limit int) DiscoveryQuery {
	oldest, youngest := profiles.BirthdateBoundsForAges(v.MinAge, v.MaxAge, today)
	return DiscoveryQuery{
		Viewer:          v,
		ViewerAge:       profiles.AgeOn(v.Birthdate, today),
		OldestBirthdate: oldest,
		YoungestBirth:   youngest,
		// 1° of latitude ≈ 111 km; small margin for snapped coordinates.
		LatDelta: float64(v.MaxDistanceKm)/111.0 + 0.05,
		Limit:    limit,
	}
}

func (s *Service) readyViewer(ctx context.Context, userID string) (Viewer, error) {
	v, ready, err := s.repo.GetViewer(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	if !ready {
		return Viewer{}, ErrProfileIncomplete
	}
	return v, nil
}

// Discover returns the next batch of profiles to review.
func (s *Service) Discover(ctx context.Context, userID string, limit int) (DeckResponse, error) {
	v, err := s.readyViewer(ctx, userID)
	if err != nil {
		return DeckResponse{}, err
	}
	_ = s.repo.TouchActivity(ctx, userID)

	ids, err := s.repo.FindCandidates(ctx, BuildQuery(v, s.now().UTC(), limit))
	if err != nil {
		return DeckResponse{}, err
	}
	cards, err := s.cards.LoadCards(ctx, userID, ids)
	if err != nil {
		return DeckResponse{}, err
	}
	if cards == nil {
		cards = []profiles.PublicProfile{}
	}
	return DeckResponse{Profiles: cards}, nil
}

// Swipe records like/pass and creates a match on reciprocal likes.
func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	if userID == targetID {
		return SwipeResult{}, ErrSelfSwipe
	}
	v, err := s.readyViewer(ctx, userID)
	if err != nil {
		return SwipeResult{}, err
	}
	stored, err := s.repo.Swipe(ctx, BuildQuery(v, s.now().UTC(), 1), targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	result := SwipeResult{Action: action}
	if stored == nil {
		return result, nil
	}

	cards, err := s.cards.LoadCards(ctx, userID, []string{userID, targetID})
	if err != nil {
		return SwipeResult{}, err
	}
	byID := map[string]profiles.PublicProfile{}
	for _, c := range cards {
		byID[c.UserID] = c
	}
	mine := summarize(*stored, byID[targetID])
	theirs := summarize(StoredMatch{ID: stored.ID, ConversationID: stored.ConversationID, OtherUserID: userID, CreatedAt: stored.CreatedAt}, byID[userID])

	result.Matched = true
	result.Match = &mine

	if s.publisher != nil {
		s.publisher.Publish([]string{userID}, realtime.Event{Type: "match.created", Data: mine})
		s.publisher.Publish([]string{targetID}, realtime.Event{Type: "match.created", Data: theirs})
	}
	if s.notifier != nil {
		data := map[string]string{"matchId": stored.ID, "conversationId": stored.ConversationID}
		// Each notification names the other person of the match.
		s.notifier.Notify(ctx, targetID, "match", "Nouveau match", "Toi et "+theirs.User.FirstName+" vous êtes plu. Lance la conversation !", withUser(data, userID))
		s.notifier.Notify(ctx, userID, "match", "Nouveau match", "Toi et "+mine.User.FirstName+" vous êtes plu. Lance la conversation !", withUser(data, targetID))
	}
	return result, nil
}

func withUser(data map[string]string, userID string) map[string]string {
	out := make(map[string]string, len(data)+1)
	for k, v := range data {
		out[k] = v
	}
	out["userId"] = userID
	return out
}

func summarize(m StoredMatch, card profiles.PublicProfile) MatchSummary {
	user := MatchUser{UserID: m.OtherUserID, FirstName: card.FirstName, Age: card.Age}
	if len(card.Photos) > 0 {
		photo := card.Photos[0]
		user.Photo = &photo
	}
	return MatchSummary{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		CreatedAt:      m.CreatedAt,
		User:           user,
		LastMessage:    m.LastMessage,
		UnreadCount:    m.UnreadCount,
	}
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit, offset int) (MatchesResponse, error) {
	stored, err := s.repo.ListMatches(ctx, userID, limit+1, offset)
	if err != nil {
		return MatchesResponse{}, err
	}
	hasMore := len(stored) > limit
	if hasMore {
		stored = stored[:limit]
	}
	ids := make([]string, 0, len(stored))
	for _, m := range stored {
		ids = append(ids, m.OtherUserID)
	}
	cards, err := s.cards.LoadCards(ctx, userID, ids)
	if err != nil {
		return MatchesResponse{}, err
	}
	byID := make(map[string]profiles.PublicProfile, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]MatchSummary, 0, len(stored))
	for _, m := range stored {
		card, ok := byID[m.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, summarize(m, card))
	}
	return MatchesResponse{Matches: out, HasMore: hasMore}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	if s.publisher != nil {
		s.publisher.Publish([]string{userID, other}, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	}
	return nil
}
