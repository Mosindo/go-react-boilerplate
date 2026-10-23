package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

type Data struct {
	MatchID string `json:"matchId,omitempty"`
	UserID  string `json:"userId,omitempty"`
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

type ListResult struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextCursor    *string        `json:"nextCursor"`
}
