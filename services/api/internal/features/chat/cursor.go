package chat

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
)

var ErrInvalidCursor = errors.New("invalid cursor")

// Cursor is an opaque keyset position (timestamp + id) used for pagination.
type Cursor struct {
	At time.Time
	ID string
}

func (c Cursor) Encode() string {
	raw := c.At.UTC().Format(time.RFC3339Nano) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || !httpx.IsUUID(parts[1]) {
		return nil, ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, ErrInvalidCursor
	}
	return &Cursor{At: at, ID: parts[1]}, nil
}
