package notifications

import (
	"encoding/json"
	"time"
)

const (
	TypeMatch      = "match"
	TypeLike       = "like"
	TypeNewMessage = "new_message"
)

type Notification struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data"`
	IsRead    bool            `json:"isRead"`
	CreatedAt time.Time       `json:"createdAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    string         `json:"nextCursor,omitempty"`
	Unread        int            `json:"unread"`
}
