// Package pagination provides opaque keyset cursors for (timestamp, id) ordered lists.
package pagination

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("invalid cursor")

func Encode(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func Decode(cursor string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	return at, parts[1], nil
}

// Limit clamps a requested page size into [1, max], using def when unset.
func Limit(requested, def, max int) int {
	if requested <= 0 {
		return def
	}
	if requested > max {
		return max
	}
	return requested
}
