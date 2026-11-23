package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDeckSize = 10
	MaxDeckSize     = 30
)

// Viewer holds the data of the person browsing, used to build the query.
type Viewer struct {
	UserID        string
	Gender        string
	Birthdate     time.Time
	Latitude      float64
	Longitude     float64
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
}

// DiscoveryQuery is the fully resolved candidate filter.
type DiscoveryQuery struct {
	Viewer
	ViewerAge       int
	OldestBirthdate time.Time
	YoungestBirth   time.Time
	LatDelta        float64
	Limit           int
	OnlyUserID      string
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type SwipeResult struct {
	Action  string        `json:"action"`
	Matched bool          `json:"matched"`
	Match   *MatchSummary `json:"match,omitempty"`
}

type StoredMatch struct {
	ID             string
	ConversationID string
	OtherUserID    string
	CreatedAt      time.Time
	LastMessage    *LastMessage
	UnreadCount    int
}

type LastMessage struct {
	Content      string    `json:"content"`
	SenderUserID string    `json:"senderUserId"`
	CreatedAt    time.Time `json:"createdAt"`
}

type MatchUser struct {
	UserID    string          `json:"userId"`
	FirstName string          `json:"firstName"`
	Age       int             `json:"age"`
	Photo     *profiles.Photo `json:"photo"`
}

type MatchSummary struct {
	ID             string       `json:"id"`
	ConversationID string       `json:"conversationId"`
	CreatedAt      time.Time    `json:"createdAt"`
	User           MatchUser    `json:"user"`
	LastMessage    *LastMessage `json:"lastMessage"`
	UnreadCount    int          `json:"unreadCount"`
}

type DeckResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type MatchesResponse struct {
	Matches []MatchSummary `json:"matches"`
	HasMore bool           `json:"hasMore"`
}
