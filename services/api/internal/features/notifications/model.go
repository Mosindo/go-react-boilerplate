package notifications

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type Item struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	MatchID   string         `json:"matchId"`
	Actor     *profiles.Card `json:"actor,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	ReadAt    *time.Time     `json:"readAt,omitempty"`
}

type Page struct {
	Notifications []Item `json:"notifications"`
	Unread        int    `json:"unread"`
	NextCursor    string `json:"nextCursor,omitempty"`
}

type Summary struct {
	UnreadNotifications int `json:"unreadNotifications"`
	UnreadMessages      int `json:"unreadMessages"`
}

type Row struct {
	ID        string
	Type      string
	MatchID   string
	ActorID   string
	CreatedAt time.Time
	ReadAt    *time.Time
}
