package httpx

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var ErrBadCursor = errors.New("invalid cursor")

// EncodeCursor builds an opaque keyset cursor from a timestamp and an id.
func EncodeCursor(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

// DecodeCursor parses a cursor; the empty string yields ok=false with no error.
func DecodeCursor(raw string) (t time.Time, id string, ok bool, err error) {
	if raw == "" {
		return time.Time{}, "", false, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, "", false, ErrBadCursor
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 || len(parts[1]) != 36 {
		return time.Time{}, "", false, ErrBadCursor
	}
	t, err = time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false, ErrBadCursor
	}
	return t, parts[1], true, nil
}
