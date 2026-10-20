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
)

// Respond writes err as a JSON error. Business errors keep their status and
// message; anything else is logged and reported as a generic 500.
func Respond(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.AsAppError(err); ok {
		c.JSON(appErr.Status, gin.H{"error": appErr.Message})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

func Fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// BindJSON decodes the body into dst and answers 400 on failure.
func BindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, http.StatusBadRequest, "invalid request")
		return false
	}
	return true
}

func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// Limit parses the ?limit= query parameter.
func Limit(c *gin.Context, def, max int) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		Fail(c, http.StatusBadRequest, "invalid limit")
		return 0, false
	}
	return n, true
}

func EncodeCursor(raw string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor returns "" for an empty cursor and ok=false for a malformed one.
func DecodeCursor(cursor string) (string, bool) {
	if cursor == "" {
		return "", true
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil || len(decoded) > 256 {
		return "", false
	}
	return string(decoded), true
}

// TimeCursor is the (timestamp, id) position used by newest-first lists.
type TimeCursor struct {
	At time.Time
	ID string
}

func EncodeTimeCursor(at time.Time, id string) string {
	return EncodeCursor(at.UTC().Format(time.RFC3339Nano) + "|" + id)
}

// DecodeTimeCursor returns (nil, true) for an empty cursor.
func DecodeTimeCursor(cursor string) (*TimeCursor, bool) {
	raw, ok := DecodeCursor(cursor)
	if !ok {
		return nil, false
	}
	if raw == "" {
		return nil, true
	}
	at, id, found := strings.Cut(raw, "|")
	if !found || id == "" {
		return nil, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return nil, false
	}
	return &TimeCursor{At: parsed, ID: id}, true
}
