// Package httpx holds small, shared HTTP helpers for Gin handlers:
// consistent error responses, request binding, path/query parsing and cursors.
package httpx

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes a JSON error. Typed application errors keep their status and message;
// anything else becomes an opaque 500 and is logged with the operation name.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		if appErr.Status >= 500 {
			logger.LogHandlerError(c, operation, appErr.Status, err)
		}
		c.AbortWithStatusJSON(appErr.Status, ErrorResponse{Error: appErr.Message, Code: appErr.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Une erreur interne est survenue.", Code: "internal_error"})
}

// BindJSON decodes the request body and reports a uniform 400 on failure.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, ErrorResponse{Error: "Requête trop volumineuse.", Code: "payload_too_large"})
			return false
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Requête invalide.", Code: "invalid_request"})
		return false
	}
	return true
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// UUIDParam validates a UUID path parameter.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Identifiant invalide.", Code: "invalid_id"})
		return "", false
	}
	return strings.ToLower(value), true
}

// Limit parses a ?limit= query parameter, clamped to [1, max].
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

// Cursor is an opaque keyset pagination position (timestamp + tie-breaking id).
type Cursor struct {
	Time time.Time
	ID   string
}

func EncodeCursor(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

// DecodeCursor parses the ?cursor= query parameter. An empty value yields (nil, true).
func DecodeCursor(c *gin.Context) (*Cursor, bool) {
	raw := strings.TrimSpace(c.Query("cursor"))
	if raw == "" {
		return nil, true
	}
	cursor, err := ParseCursor(raw)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Curseur de pagination invalide.", Code: "invalid_cursor"})
		return nil, false
	}
	return cursor, true
}

func ParseCursor(raw string) (*Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 || !IsUUID(parts[1]) {
		return nil, errors.New("malformed cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, err
	}
	return &Cursor{Time: t, ID: parts[1]}, nil
}
