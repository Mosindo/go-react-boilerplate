package chat

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
)

func encodeCursor(at time.Time, id string) string {
	raw := strconv.FormatInt(at.UnixNano(), 10) + ":" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(value string) (*cursor, bool) {
	if value == "" {
		return nil, true
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, false
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 || !httpx.IsUUID(parts[1]) {
		return nil, false
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, false
	}
	return &cursor{At: time.Unix(0, nanos).UTC(), ID: parts[1]}, true
}
