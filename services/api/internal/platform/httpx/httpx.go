// Package httpx holds small Gin helpers shared by feature handlers.
package httpx

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Fail renders an expected error as JSON, or logs and hides any unexpected one behind a generic 500.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		c.AbortWithStatusJSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal"})
}

// BindJSON decodes the body into dst, answering 400 itself on failure.
func BindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(dst); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request", "code": "invalid_request"})
		return false
	}
	return true
}

func UserID(c *gin.Context) string { return c.GetString("userID") }

// UUIDParam parses a path parameter as UUID, answering 404 (not 400) so malformed ids leak nothing.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := c.Param(name)
	if _, err := uuid.Parse(value); err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found", "code": "not_found"})
		return "", false
	}
	return strings.ToLower(value), true
}

// Limit parses ?limit= with a default and maximum; invalid values fall back to the default.
func Limit(c *gin.Context, def, max int) int {
	raw := c.Query("limit")
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// Cursor is a (timestamp, id) keyset position.
type Cursor struct {
	At time.Time
	ID string
}

func EncodeCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

// DecodeCursor returns (nil, nil) for an empty cursor and an error for a malformed one.
func DecodeCursor(raw string) (*Cursor, error) {
	if raw == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, platformerrors.BadRequest("invalid_cursor", "invalid cursor")
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return nil, platformerrors.BadRequest("invalid_cursor", "invalid cursor")
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, platformerrors.BadRequest("invalid_cursor", "invalid cursor")
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return nil, platformerrors.BadRequest("invalid_cursor", "invalid cursor")
	}
	return &Cursor{At: at, ID: parts[1]}, nil
}
