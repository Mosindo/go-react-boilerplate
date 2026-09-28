package realtime

import "encoding/json"

const (
	CloseUnauthorized = 4401
	CloseTooMany      = 4429

	MaxConnsPerUser = 5
)

// Envelope is the shape of every server -> client frame.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type clientFrame struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}
