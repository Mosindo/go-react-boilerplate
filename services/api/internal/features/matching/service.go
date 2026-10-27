package matching

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
)

var (
	ErrInvalidAction     = errors.New("action must be 'like' or 'pass'")
	ErrInvalidTarget     = errors.New("invalid target")
	ErrTargetUnavailable = errors.New("profile unavailable")
	ErrAlreadySwiped     = errors.New("profile already handled")
	ErrNotFound          = errors.New("not found")
	ErrProfileRequired   = errors.New("complete your profile and add a photo first")
	ErrInvalidCursor     = errors.New("invalid cursor")
)

// Notifier is told about new matches (persisted notification + realtime push).
type Notifier interface {
	Matched(ctx context.Context, matchID, conversationID, userA, userB string)
}

// Activity records that a user is active, feeding recommendation freshness.
type Activity interface {
	Touch(ctx context.Context, userID string) error
}

type Service struct {
	repo     Repository
	notifier Notifier
	activity Activity
}

func NewService(repo Repository, notifier Notifier, activity Activity) *Service {
	return &Service{repo: repo, notifier: notifier, activity: activity}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	if limit < 1 {
		limit = DefaultDiscoverLimit
	}
	if limit > MaxDiscoverLimit {
		limit = MaxDiscoverLimit
	}
	if s.activity != nil {
		_ = s.activity.Touch(ctx, userID)
	}
	return s.repo.Candidates(ctx, userID, limit)
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	target := strings.ToLower(strings.TrimSpace(req.TargetID))
	if !httpx.IsUUID(target) || target == userID {
		return SwipeResult{}, ErrInvalidTarget
	}
	result, err := s.repo.Swipe(ctx, userID, target, action)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepoTargetUnavailable):
			return SwipeResult{}, ErrTargetUnavailable
		case errors.Is(err, ErrRepoAlreadySwiped):
			return SwipeResult{}, ErrAlreadySwiped
		case errors.Is(err, ErrRepoProfileMissing):
			return SwipeResult{}, ErrProfileRequired
		}
		return SwipeResult{}, err
	}
	if result.Matched && s.notifier != nil {
		s.notifier.Matched(ctx, result.MatchID, result.ConversationID, userID, target)
	}
	return result, nil
}

func (s *Service) Matches(ctx context.Context, userID string, limit int, cursor string) (MatchPage, error) {
	if limit < 1 || limit > httpx.MaxLimit {
		limit = httpx.DefaultLimit
	}
	var before *Cursor
	if cursor != "" {
		c, err := decodeCursor(cursor)
		if err != nil {
			return MatchPage{}, ErrInvalidCursor
		}
		before = &c
	}
	matches, next, err := s.repo.ListMatches(ctx, userID, limit, before)
	if err != nil {
		return MatchPage{}, err
	}
	page := MatchPage{Matches: matches}
	if next != nil {
		page.NextCursor = encodeCursor(*next)
	}
	return page, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	if _, err := s.repo.Unmatch(ctx, userID, matchID); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (profiles.PublicProfile, error) {
	p, err := s.repo.GetPublicProfile(ctx, viewerID, targetID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return profiles.PublicProfile{}, ErrNotFound
		}
		return profiles.PublicProfile{}, err
	}
	return p, nil
}

func encodeCursor(c Cursor) string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID))
}

func decodeCursor(raw string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Cursor{}, err
	}
	ts, id, ok := strings.Cut(string(b), "|")
	if !ok || !httpx.IsUUID(id) {
		return Cursor{}, ErrInvalidCursor
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return Cursor{}, err
	}
	return Cursor{CreatedAt: t, ID: id}, nil
}
