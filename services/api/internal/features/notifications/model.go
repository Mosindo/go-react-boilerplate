package notifications

import (
	"encoding/json"
	"time"
)

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Notification struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Read      bool            `json:"read"`
	CreatedAt time.Time       `json:"createdAt"`
}
