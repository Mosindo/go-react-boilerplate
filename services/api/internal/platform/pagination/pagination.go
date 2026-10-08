// Package pagination provides opaque keyset cursors and limit parsing for list endpoints.
package pagination

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
)

// Limit parses a ?limit= value, applying def when empty and rejecting out-of-range values.
func Limit(raw string, def, max int) (int, error) {
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		return 0, platformerrors.BadRequest("invalid_limit", "limit must be between 1 and "+strconv.Itoa(max))
	}
	return n, nil
}

// Cursor is a (timestamp, id) keyset position.
type Cursor struct {
	At time.Time
	ID string
}

func (c Cursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.At.UTC().Format(time.RFC3339Nano) + "|" + c.ID))
}

// Decode parses a cursor; an empty string yields (nil, nil).
func Decode(raw string) (*Cursor, error) {
	if raw == "" {
		return nil, nil
	}
	bad := platformerrors.BadRequest("invalid_cursor", "invalid cursor")
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, bad
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 || !platformerrors.IsUUID(parts[1]) {
		return nil, bad
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, bad
	}
	return &Cursor{At: at, ID: parts[1]}, nil
}
