package matching

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = platformerrors.Conflict("profile_incomplete", "Complétez votre profil (infos, préférences et au moins une photo) pour découvrir des profils.")
	ErrInvalidAction     = platformerrors.Validation("invalid_action", "Action invalide.")
	ErrSelfSwipe         = platformerrors.Validation("invalid_target", "Action impossible sur votre propre profil.")
	ErrAlreadySwiped     = platformerrors.Conflict("already_swiped", "Vous avez déjà répondu à ce profil.")
	ErrTargetUnavailable = platformerrors.NotFound("profile_not_found", "Ce profil n'est plus disponible.")
	ErrMatchNotFound     = platformerrors.NotFound("match_not_found", "Match introuvable.")
)

type Notifier interface {
	Notify(ctx context.Context, in notifications.Input)
	Forget(ctx context.Context, groupKeys ...string)
	MarkGroupRead(ctx context.Context, userID, groupKey string)
}

type Service struct {
	repo      Repository
	profiles  *profiles.Service
	ranker    Ranker
	notifier  Notifier
	publisher realtime.Publisher
	now       func() time.Time
}

func NewService(repo Repository, profileService *profiles.Service, ranker Ranker, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profileService, ranker: ranker, notifier: notifier, publisher: publisher, now: time.Now}
}

func (s *Service) requireComplete(ctx context.Context, userID string) (viewer, error) {
	v, err := s.repo.Viewer(ctx, userID)
	if err != nil {
		return viewer{}, err
	}
	if !v.Complete {
		return viewer{}, ErrProfileIncomplete
	}
	return v, nil
}

// Deck returns the next profiles to review, best first.
func (s *Service) Deck(ctx context.Context, userID string, size int) (DeckResponse, error) {
	if size <= 0 || size > maxDeckSize {
		size = defaultDeckSize
	}
	v, err := s.requireComplete(ctx, userID)
	if err != nil {
		return DeckResponse{}, err
	}
	candidates, err := s.repo.Candidates(ctx, v, candidatePool)
	if err != nil {
		return DeckResponse{}, err
	}
	ranked := s.ranker.Rank(candidates, v.MaxDistanceKm, s.now())
	if len(ranked) > size {
		ranked = ranked[:size]
	}
	ids := make([]string, len(ranked))
	for i, c := range ranked {
		ids[i] = c.UserID
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return DeckResponse{}, err
	}
	deck := DeckResponse{Profiles: make([]profiles.PublicProfile, 0, len(ids))}
	for _, id := range ids {
		if card, ok := cards[id]; ok {
			deck.Profiles = append(deck.Profiles, card)
		}
	}
	if len(deck.Profiles) == 0 {
		deck.Reason = "no_more_profiles"
		if v.Latitude == nil {
			deck.Reason = "no_more_profiles_location_off"
		}
	}
	return deck, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	action := strings.TrimSpace(req.Action)
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	targetID := strings.ToLower(strings.TrimSpace(req.TargetUserID))
	if !httpx.IsUUID(targetID) {
		return SwipeResult{}, ErrTargetUnavailable
	}
	if targetID == userID {
		return SwipeResult{}, ErrSelfSwipe
	}
	if _, err := s.requireComplete(ctx, userID); err != nil {
		return SwipeResult{}, err
	}

	outcome, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		if errors.Is(err, errRepoTargetUnavailable) {
			return SwipeResult{}, ErrTargetUnavailable
		}
		return SwipeResult{}, err
	}
	if outcome.AlreadySwiped {
		return SwipeResult{}, ErrAlreadySwiped
	}

	result := SwipeResult{Action: action}
	if action == ActionLike && !outcome.Matched {
		s.notifier.Notify(ctx, notifications.Input{
			UserID:   targetID,
			Type:     notifications.TypeLike,
			Title:    "Quelqu'un s'intéresse à vous",
			Body:     "Découvrez qui vous a liké dans l'onglet Likes.",
			Data:     map[string]any{},
			GroupKey: likesGroupKey,
		})
		return result, nil
	}
	if !outcome.Matched {
		return result, nil
	}

	cards, err := s.profiles.Cards(ctx, userID, []string{targetID})
	if err != nil {
		return SwipeResult{}, err
	}
	// Each side sees the other's card, with distance computed from their own position.
	targetCards, err := s.profiles.Cards(ctx, targetID, []string{userID})
	if err != nil {
		return SwipeResult{}, err
	}
	forMe := MatchView{ID: outcome.MatchID, ConversationID: outcome.ConversationID, CreatedAt: outcome.CreatedAt, Profile: cards[targetID]}
	forThem := MatchView{ID: outcome.MatchID, ConversationID: outcome.ConversationID, CreatedAt: outcome.CreatedAt, Profile: targetCards[userID]}

	s.announceMatch(ctx, userID, forMe)
	s.announceMatch(ctx, targetID, forThem)

	result.Matched = true
	result.Match = &forMe
	return result, nil
}

func (s *Service) announceMatch(ctx context.Context, userID string, m MatchView) {
	s.publisher.Publish(userID, realtime.Event{Type: "match.created", Data: m})
	s.notifier.Notify(ctx, notifications.Input{
		UserID:   userID,
		Type:     notifications.TypeMatch,
		Title:    "Nouveau match !",
		Body:     "Vous et " + m.Profile.FirstName + " vous plaisez mutuellement. Lancez la conversation.",
		Data:     map[string]any{"matchId": m.ID, "conversationId": m.ConversationID, "userId": m.Profile.UserID},
		GroupKey: "match:" + m.ID,
	})
}

func (s *Service) ListMatches(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) (MatchesPage, error) {
	rows, err := s.repo.ListMatches(ctx, userID, cursor, limit+1)
	if err != nil {
		return MatchesPage{}, err
	}
	page := MatchesPage{}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next := httpx.EncodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	page.Matches, err = s.views(ctx, userID, rows)
	return page, err
}

func (s *Service) views(ctx context.Context, userID string, rows []matchRow) ([]MatchView, error) {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.OtherUserID
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]MatchView, 0, len(rows))
	for _, r := range rows {
		card, ok := cards[r.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, MatchView{ID: r.ID, ConversationID: r.ConversationID, CreatedAt: r.CreatedAt, HasMessages: r.HasMessages, Profile: card})
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	m, err := s.repo.DeleteMatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, errRepoNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	s.announceRemoval(ctx, userID, m)
	return nil
}

// RemoveMatchBetween is used by safety features (block, report).
func (s *Service) RemoveMatchBetween(ctx context.Context, userID, otherID string) error {
	m, err := s.repo.DeleteMatchBetween(ctx, userID, otherID)
	if err != nil || m == nil {
		return err
	}
	s.announceRemoval(ctx, userID, *m)
	return nil
}

func (s *Service) announceRemoval(ctx context.Context, userID string, m matchRow) {
	s.notifier.Forget(ctx, "match:"+m.ID, "conversation:"+m.ConversationID)
	payload := map[string]string{"matchId": m.ID, "conversationId": m.ConversationID}
	s.publisher.Publish(userID, realtime.Event{Type: "match.removed", Data: payload})
	s.publisher.Publish(m.OtherUserID, realtime.Event{Type: "match.removed", Data: payload})
}

// likesGroupKey coalesces "someone liked you" notifications into a single unread item.
const likesGroupKey = "likes"

// LikesReceived lists members who liked the user; viewing the list marks the like notification as seen.
func (s *Service) LikesReceived(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	ids, err := s.repo.LikesReceived(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	s.notifier.MarkGroupRead(ctx, userID, likesGroupKey)
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.PublicProfile, 0, len(ids))
	for _, id := range ids {
		if c, ok := cards[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}
