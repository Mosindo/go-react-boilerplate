// Package cursor encodes opaque keyset-pagination cursors: base64("<RFC3339Nano>|<uuid>").
package cursor

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid cursor")

type Position struct {
	Time time.Time
	ID   string
}

func Encode(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

// Decode returns nil for an empty cursor (first page).
func Decode(raw string) (*Position, error) {
	if raw == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrInvalid
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 || len(parts[1]) != 36 {
		return nil, ErrInvalid
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, ErrInvalid
	}
	return &Position{Time: t, ID: parts[1]}, nil
}
