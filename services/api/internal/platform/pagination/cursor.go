// Package pagination implements opaque keyset cursors (timestamp + id).
package pagination

import (
	"encoding/base64"
	"strings"
	"time"
)

// Encode builds an opaque cursor for the last item of a page.
func Encode(t time.Time, id string) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// Decode parses a cursor; ok is false for anything malformed.
func Decode(cursor string) (t time.Time, id string, ok bool) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", false
	}
	ts, rest, found := strings.Cut(string(raw), "|")
	if !found || rest == "" {
		return time.Time{}, "", false
	}
	t, err = time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, "", false
	}
	return t, rest, true
}
