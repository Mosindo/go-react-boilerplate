package matches

import (
	"time"

	"example.com/api/internal/features/profiles"
)

// Item is a match as seen by one of the two users.
type Item struct {
	MatchID        string                 `json:"matchId"`
	ConversationID string                 `json:"conversationId"`
	CreatedAt      time.Time              `json:"createdAt"`
	Profile        profiles.PublicProfile `json:"profile"`
}

type row struct {
	MatchID        string
	ConversationID string
	CreatedAt      time.Time
	OtherUserID    string
}
