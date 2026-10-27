package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Actor struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	PhotoID   string `json:"photoId,omitempty"`
}

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Actor     *Actor     `json:"actor,omitempty"`
	RefID     string     `json:"refId,omitempty"` // the conversation the notification leads to
	ReadAt    *time.Time `json:"readAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type Page struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextCursor    string         `json:"nextCursor,omitempty"`
}
