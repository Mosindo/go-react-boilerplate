package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeLike    = "like"
	TypeMessage = "message"
)

type Notification struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data"`
	IsRead    bool           `json:"isRead"`
	CreatedAt time.Time      `json:"createdAt"`
}

type Page struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *string        `json:"nextCursor"`
	UnreadCount   int            `json:"unreadCount"`
}

// Input describes a notification to create. Unread notifications with the same GroupKey
// are coalesced (the existing row is refreshed instead of adding a new one).
type Input struct {
	UserID   string
	Type     string
	Title    string
	Body     string
	Data     map[string]any
	GroupKey string
}
