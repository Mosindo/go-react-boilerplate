// Package httpx holds small helpers shared by feature handlers: error responses,
// path/pagination parsing and strict JSON binding.
package httpx

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func IsUUID(value string) bool { return uuidPattern.MatchString(value) }

// Fail logs the underlying error and answers with a stable JSON error payload.
func Fail(c *gin.Context, operation string, status int, message string, err error) {
	if err != nil {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// Internal answers 500 without leaking internals.
func Internal(c *gin.Context, operation string, err error) {
	Fail(c, operation, http.StatusInternalServerError, "internal error", err)
}

// BindJSON decodes the body into dst and answers 400 on failure.
func BindJSON(c *gin.Context, operation string, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, operation+".bind", http.StatusBadRequest, "invalid request", err)
		return false
	}
	return true
}

// UUIDParam reads a path parameter and answers 404 when it is not a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return "", false
	}
	return strings.ToLower(value), true
}

// Pagination parses ?limit&offset and answers 400 when they are not valid integers.
func Pagination(c *gin.Context, defaultLimit, maxLimit int) (limit, offset int, ok bool) {
	limit, offset = defaultLimit, 0
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return 0, 0, false
		}
		limit = v
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 || v > 100000 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return 0, 0, false
		}
		offset = v
	}
	return limit, offset, true
}
