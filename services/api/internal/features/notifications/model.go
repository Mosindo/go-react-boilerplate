package notifications

import (
	"encoding/json"
	"time"
)

const (
	TypeMatch  = "match"
	TypeSystem = "system"
)

type Notification struct {
	ID        string          `json:"id"`
	UserID    string          `json:"-"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data"`
	ReadAt    *time.Time      `json:"readAt"`
	CreatedAt time.Time       `json:"createdAt"`
}

// New describes a notification to create.
type New struct {
	UserID string
	Type   string
	Title  string
	Body   string
	Data   map[string]string
}
