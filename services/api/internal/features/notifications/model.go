package notifications

import "time"

const TypeMatch = "match"

type Notification struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data"`
	IsRead    bool           `json:"isRead"`
	CreatedAt time.Time      `json:"createdAt"`
	ReadAt    *time.Time     `json:"readAt,omitempty"`
}

// New describes a notification to create; Data carries ids the client needs to deep-link.
type New struct {
	Type  string
	Title string
	Body  string
	Data  map[string]any
}

// Notifier is the narrow dependency other features use to notify users.
type Notifier interface {
	Notify(userID string, n New)
}
