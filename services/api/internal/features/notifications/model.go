package notifications

import "time"

const (
	TypeNewLike  = "new_like"
	TypeNewMatch = "new_match"
)

type Notification struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	IsRead    bool              `json:"isRead"`
	CreatedAt time.Time         `json:"createdAt"`
}

type NewNotification struct {
	UserID string
	Type   string
	Title  string
	Body   string
	Data   map[string]string
}
