package notifications

import (
	"encoding/json"
	"time"
)

const (
	TypeMatch   = "match"
	TypeMessage = "message"
	TypeLike    = "like"
)

type Notification struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data"`
	ReadAt    *time.Time      `json:"readAt"`
	CreatedAt time.Time       `json:"createdAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *string        `json:"nextCursor"`
}

// Input describes a notification produced by another feature.
type Input struct {
	UserID string
	Type   string
	Title  string
	Body   string
	Data   map[string]string
	// GroupKey collapses repeated unread notifications into one row.
	GroupKey string
}

type Badges struct {
	UnreadNotifications int `json:"unreadNotifications"`
	UnreadConversations int `json:"unreadConversations"`
	LikesReceived       int `json:"likesReceived"`
}
