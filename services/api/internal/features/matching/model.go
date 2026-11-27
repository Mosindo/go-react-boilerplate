package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDiscoveryLimit = 10
	MaxDiscoveryLimit     = 20
	// CandidatePoolSize bounds the SQL result handed to the ranker.
	CandidatePoolSize = 100

	DefaultMatchesLimit = 20
	MaxMatchesLimit     = 50
)

// Candidate is a discoverable profile with the signals used for ranking.
type Candidate struct {
	UserID          string
	LastActiveAt    time.Time
	DistanceKm      *float64
	SharedInterests int
	LikedViewer     bool
}

type SwipeResult struct {
	Action        string
	AlreadySwiped bool
	Matched       bool
	MatchID       string
	OtherUserID   string
}

type LastMessage struct {
	ID           string
	SenderUserID string
	Content      string
	CreatedAt    time.Time
}

type MatchSummary struct {
	ID          string
	OtherUserID string
	MatchedAt   time.Time
	ActivityAt  time.Time
	LastMessage *LastMessage
	UnreadCount int
}

// --- HTTP payloads ---------------------------------------------------------

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type SwipeResponse struct {
	Action        string                `json:"action"`
	AlreadySwiped bool                  `json:"alreadySwiped"`
	Matched       bool                  `json:"matched"`
	Match         *MatchSummaryResponse `json:"match,omitempty"`
}

type DiscoveryResponse struct {
	Profiles []profiles.CardResponse `json:"profiles"`
}

type LastMessageResponse struct {
	ID           string    `json:"id"`
	SenderUserID string    `json:"senderUserId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
}

type MatchSummaryResponse struct {
	ID          string                `json:"id"`
	MatchedAt   time.Time             `json:"matchedAt"`
	User        profiles.CardResponse `json:"user"`
	LastMessage *LastMessageResponse  `json:"lastMessage,omitempty"`
	UnreadCount int                   `json:"unreadCount"`
}

type MatchesResponse struct {
	Matches    []MatchSummaryResponse `json:"matches"`
	NextCursor string                 `json:"nextCursor,omitempty"`
}
