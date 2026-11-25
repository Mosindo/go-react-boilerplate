// Package httpx contains small, shared helpers used by Gin handlers:
// consistent error responses, identity extraction and query parsing.
package httpx

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

const userIDKey = "userID"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ErrorBody is the JSON shape of every error response.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes an error response. Typed application errors are exposed as-is;
// any other error is logged and hidden behind a generic 500.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := apperr.As(err); ok {
		if appErr.Status >= http.StatusInternalServerError {
			logger.LogHandlerError(c, operation, appErr.Status, err)
		}
		c.AbortWithStatusJSON(appErr.Status, ErrorBody{Error: appErr.Message, Code: appErr.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorBody{Error: "internal server error", Code: "internal_error"})
}

// BindJSON decodes the request body and answers 400 on failure.
func BindJSON(c *gin.Context, operation string, dst any) bool {
	// JSON endpoints only accept JSON: this also guarantees the global body
	// size limit (skipped for multipart uploads) applies to them.
	if c.ContentType() != "application/json" {
		c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, ErrorBody{Error: "content type must be application/json", Code: "unsupported_media_type"})
		return false
	}
	if err := c.ShouldBindJSON(dst); err != nil {
		logger.LogHandlerError(c, operation+".bind", http.StatusBadRequest, err)
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorBody{Error: "invalid request body", Code: "invalid_request"})
		return false
	}
	return true
}

// SetUserID stores the authenticated user id on the context.
func SetUserID(c *gin.Context, userID string) {
	c.Set(userIDKey, userID)
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString(userIDKey)
}

func IsUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

// UUIDParam reads a path parameter and validates it is a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorBody{Error: "invalid " + name, Code: "invalid_id"})
		return "", false
	}
	return strings.ToLower(value), true
}

// Limit parses a bounded `limit` query parameter.
func Limit(c *gin.Context, fallback, max int) int {
	return ClampInt(c.Query("limit"), fallback, 1, max)
}

// Offset parses a non-negative `offset` query parameter.
func Offset(c *gin.Context) int {
	return ClampInt(c.Query("offset"), 0, 0, 10_000)
}

// ClampInt parses raw as an int and clamps it to [min, max]. Invalid input
// yields fallback.
func ClampInt(raw string, fallback, min, max int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}
