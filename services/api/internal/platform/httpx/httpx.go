// Package httpx holds small Gin helpers shared by every feature handler:
// consistent error payloads, authenticated user lookup and input parsing.
package httpx

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/platform/apperr"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorResponse is the single error shape returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

var (
	ErrInvalidRequest = apperr.BadRequest("invalid_request", "invalid request")
	ErrInvalidID      = apperr.BadRequest("invalid_id", "invalid identifier")
	ErrInvalidCursor  = apperr.BadRequest("invalid_cursor", "invalid cursor")
)

// Fail writes the response for err. Application errors keep their status and
// message; anything else becomes an opaque 500 and is logged.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := apperr.As(err); ok {
		if appErr.Status >= 500 {
			logger.LogHandlerError(c, operation, appErr.Status, err)
		}
		c.AbortWithStatusJSON(appErr.Status, ErrorResponse{Error: appErr.Message, Code: appErr.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "internal error", Code: "internal"})
}

// Bind decodes a JSON body and writes a 400 on failure.
func Bind(c *gin.Context, operation string, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		logger.LogHandlerError(c, operation+".bind", http.StatusBadRequest, err)
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: ErrInvalidRequest.Message, Code: ErrInvalidRequest.Code})
		return false
	}
	return true
}

// UserID returns the authenticated user id set by middleware.RequireUser.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// UUIDParam reads and validates a UUID path parameter.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(value) {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: ErrInvalidID.Message, Code: ErrInvalidID.Code})
		return "", false
	}
	return value, true
}

// IsUUID validates the canonical 8-4-4-4-12 hex representation.
func IsUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i := 0; i < len(v); i++ {
		ch := v[i]
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}
	return true
}

// Limit parses a "limit" query parameter clamped to [1, max].
func Limit(c *gin.Context, fallback, max int) int {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

// Cursor is an opaque keyset pagination position (timestamp + id tiebreaker).
type Cursor struct {
	At time.Time
	ID string
}

func EncodeCursor(at time.Time, id string) string {
	raw := at.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses the "cursor" (or provided) query parameter. A missing
// value returns nil without error.
func DecodeCursor(raw string) (*Cursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if len(raw) > 200 {
		return nil, ErrInvalidCursor
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 || !IsUUID(parts[1]) {
		return nil, ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, ErrInvalidCursor
	}
	return &Cursor{At: at, ID: parts[1]}, nil
}

// NoStore marks a response as non-cacheable (tokens, private data).
func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}
