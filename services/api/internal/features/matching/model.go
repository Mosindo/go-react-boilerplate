package matching

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadySwiped  = errors.New("already swiped")
	ErrInvalidAction  = errors.New("invalid action")
	ErrProfileMissing = errors.New("profile required")
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultFeedSize = 10
	MaxFeedSize     = 20
)

// Card est un profil propose en decouverte.
type Card struct {
	UserID          string              `json:"userId"`
	FirstName       string              `json:"firstName"`
	Age             int                 `json:"age"`
	Gender          string              `json:"gender"`
	Bio             string              `json:"bio"`
	City            string              `json:"city"`
	DistanceKm      *int                `json:"distanceKm"`
	Interests       []profiles.Interest `json:"interests"`
	SharedInterests int                 `json:"sharedInterests"`
	PhotoIDs        []string            `json:"photoIds"`
	LikedYou        bool                `json:"-"`
	distanceExactKm float64
	showDistance    bool
}

type SwipeResult struct {
	Action         string `json:"action"`
	Matched        bool   `json:"matched"`
	MatchID        string `json:"matchId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

// MatchSummary est un match dans la liste des matchs.
type MatchSummary struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	CreatedAt      time.Time `json:"createdAt"`
	UserID         string    `json:"userId"`
	FirstName      string    `json:"firstName"`
	Age            int       `json:"age"`
	PhotoID        *string   `json:"photoId"`
	HasMessages    bool      `json:"hasMessages"`
}

// Notification pousse en temps reel lors d'un match.
type MatchEvent struct {
	MatchID        string `json:"matchId"`
	ConversationID string `json:"conversationId"`
	UserID         string `json:"userId"`
	FirstName      string `json:"firstName"`
	PhotoID        string `json:"photoId,omitempty"`
}
