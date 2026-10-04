// Package pagination implements opaque keyset cursors (timestamp + id).
package pagination

import (
	"encoding/base64"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/google/uuid"
)

// Cursor marks the last row of a page when ordering by (time DESC, id DESC).
type Cursor struct {
	Time time.Time
	ID   string
}

func Encode(t time.Time, id string) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// Decode returns nil for an empty cursor.
func Decode(value string) (*Cursor, error) {
	if value == "" {
		return nil, nil
	}
	invalid := platformerrors.FieldInvalid("cursor", "invalid cursor")
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, invalid
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, invalid
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, invalid
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return nil, invalid
	}
	return &Cursor{Time: t, ID: parts[1]}, nil
}
