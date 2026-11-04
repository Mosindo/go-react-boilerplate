// Package httpx contains small helpers shared by all HTTP handlers.
package httpx

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorBody is the single error payload shape returned by the API.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes err as an HTTP error. Unknown errors become an opaque 500.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		if appErr.Status >= 500 {
			logger.LogHandlerError(c, operation, appErr.Status, err)
		}
		c.AbortWithStatusJSON(appErr.Status, ErrorBody{Error: appErr.Message, Code: appErr.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorBody{Error: "internal error", Code: "internal"})
}

// BadRequest writes a 400 with the given message.
func BadRequest(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorBody{Error: message, Code: code})
}

// BindJSON decodes the body (max 64 KiB) and writes a 400 on failure.
func BindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	if err := c.ShouldBindJSON(dst); err != nil {
		BadRequest(c, "invalid_request", "invalid request")
		return false
	}
	return true
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string { return c.GetString("userID") }

// Limit parses the ?limit= query parameter, clamped to [1, max].
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

// NoStore marks the response as non cacheable (private data).
func NoStore(c *gin.Context) { c.Header("Cache-Control", "no-store") }

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical UUID string.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }

// ParamUUID reads a path parameter and writes a 404 when it is not a UUID.
func ParamUUID(c *gin.Context, name string) (string, bool) {
	v := c.Param(name)
	if !IsUUID(v) {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorBody{Error: "not found", Code: "not_found"})
		return "", false
	}
	return strings.ToLower(v), true
}
