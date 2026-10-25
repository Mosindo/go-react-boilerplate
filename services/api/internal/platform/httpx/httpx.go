// Package httpx holds the small HTTP helpers shared by every feature handler.
package httpx

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

// Fail writes the standard error payload: {"error": message, "code": code}.
func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": message, "code": code})
}

func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, "invalid_request", message)
}

func NotFound(c *gin.Context) {
	Fail(c, http.StatusNotFound, "not_found", "not found")
}

func Internal(c *gin.Context) {
	Fail(c, http.StatusInternalServerError, "internal", "internal server error")
}

// Limit parses ?limit= with a default and a maximum.
func Limit(c *gin.Context, def, max int) (int, error) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		return 0, errors.New("limit must be between 1 and " + strconv.Itoa(max))
	}
	return n, nil
}

// Page is the envelope used by every paginated list.
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// Cursor is a keyset position (timestamp + id) encoded as an opaque string.
type Cursor struct {
	At time.Time
	ID string
}

func (c Cursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.At.UTC().Format(time.RFC3339Nano) + "|" + c.ID))
}

// DecodeCursor returns (nil, nil) for an empty string.
func DecodeCursor(raw string) (*Cursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, errors.New("invalid cursor")
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return nil, errors.New("invalid cursor")
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, errors.New("invalid cursor")
	}
	return &Cursor{At: at, ID: parts[1]}, nil
}

// QueryCursor decodes ?cursor= and writes a 400 on failure (ok=false).
func QueryCursor(c *gin.Context) (*Cursor, bool) {
	cur, err := DecodeCursor(c.Query("cursor"))
	if err != nil {
		BadRequest(c, "invalid cursor")
		return nil, false
	}
	return cur, true
}

// IsUUID reports whether s looks like a canonical UUID, so malformed path ids are rejected before reaching SQL.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}

// UUIDParam reads a path parameter and writes a 404 if it is not a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	v := strings.ToLower(c.Param(name))
	if !IsUUID(v) {
		NotFound(c)
		return "", false
	}
	return v, true
}

// Paginate trims the lookahead row and builds the page. Rows must have been fetched with limit+1.
func Paginate[T any](rows []T, limit int, cursorOf func(T) Cursor) Page[T] {
	page := Page[T]{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		next := cursorOf(rows[limit-1]).Encode()
		page.NextCursor = &next
	}
	if page.Items == nil {
		page.Items = []T{}
	}
	return page
}
