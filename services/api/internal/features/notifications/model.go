package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

// Data is the small structured payload clients use to deep-link.
type Data struct {
	MatchID        string `json:"matchId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
	UserID         string `json:"userId,omitempty"`
}

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Data      Data       `json:"data"`
	ReadAt    *time.Time `json:"readAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type ListResponse struct {
	Items       []Notification `json:"items"`
	NextCursor  *string        `json:"nextCursor"`
	UnreadCount int            `json:"unreadCount"`
}

// NewNotification is the input of the transactional creators used by other
// features (swipes, chat) inside their own transactions.
type NewNotification struct {
	UserID string
	Type   string
	Title  string
	Body   string
	Data   Data
}
