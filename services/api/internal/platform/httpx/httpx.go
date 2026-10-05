// Package httpx holds small helpers shared by Gin handlers so feature handlers stay thin.
package httpx

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Fail logs err and writes the standard error payload {"error": message, "code": code}.
func Fail(c *gin.Context, operation string, status int, code, message string, err error) {
	if err != nil {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}

// Invalid writes a 400 "invalid_request" error.
func Invalid(c *gin.Context, operation string, err error) {
	Fail(c, operation, http.StatusBadRequest, "invalid_request", "invalid request", err)
}

// Internal writes a 500 error without leaking details.
func Internal(c *gin.Context, operation string, err error) {
	Fail(c, operation, http.StatusInternalServerError, "internal_error", "internal error", err)
}

// BindJSON decodes the request body and reports failures as 400. It returns false when it already responded.
func BindJSON(c *gin.Context, operation string, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(dst); err != nil {
		Invalid(c, operation+".bind", err)
		return false
	}
	return true
}

func IsUUID(v string) bool { return uuidPattern.MatchString(v) }

// UUIDParam reads a path parameter and rejects anything that is not a UUID.
func UUIDParam(c *gin.Context, operation, name string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(value) {
		Fail(c, operation, http.StatusBadRequest, "invalid_id", "invalid "+name, nil)
		return "", false
	}
	return value, true
}

// Limit parses a "limit" style query value into [1, max], falling back to def.
func Limit(raw string, def, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// Offset parses a non-negative offset, falling back to 0.
func Offset(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 {
		return 0
	}
	return n
}
