// Package httpx holds small helpers shared by every Gin handler so that
// errors, pagination and identifiers are handled the same way everywhere.
package httpx

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorResponse is the single error shape returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Fail writes an error response and aborts the request.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: code})
}

// FailInternal logs the underlying error and returns a generic 500.
func FailInternal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, "internal_error", "something went wrong, please try again")
}

// BindJSON decodes a JSON body and replies 400 on failure.
func BindJSON(c *gin.Context, operation string, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		logger.LogHandlerError(c, operation+".bind", http.StatusBadRequest, err)
		Fail(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return false
	}
	return true
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// IsUUID reports whether value is a canonical UUID string.
func IsUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

// PathUUID reads a UUID path parameter and replies 400 when it is malformed.
func PathUUID(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		Fail(c, http.StatusBadRequest, "invalid_id", "invalid identifier")
		return "", false
	}
	return strings.ToLower(value), true
}

var errInvalidInt = errors.New("invalid integer")

func parseInt(raw string, fallback, min, max int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < min || value > max {
		return 0, errInvalidInt
	}
	return value, nil
}

// QueryLimit parses ?limit=, rejecting anything outside [1, max].
func QueryLimit(c *gin.Context, fallback, max int) (int, bool) {
	value, err := parseInt(c.Query("limit"), fallback, 1, max)
	if err != nil {
		Fail(c, http.StatusBadRequest, "invalid_pagination", "limit must be an integer between 1 and "+strconv.Itoa(max))
		return 0, false
	}
	return value, true
}

// QueryOffset parses ?offset=, rejecting negative or huge values.
func QueryOffset(c *gin.Context) (int, bool) {
	value, err := parseInt(c.Query("offset"), 0, 0, 10_000)
	if err != nil {
		Fail(c, http.StatusBadRequest, "invalid_pagination", "offset must be an integer between 0 and 10000")
		return 0, false
	}
	return value, true
}
