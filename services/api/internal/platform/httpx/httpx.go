// Package httpx holds small helpers shared by all feature handlers so that
// error responses and request parsing stay consistent across the API.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Fail writes the standard error envelope: {"error": "<message>", "code": "<slug>"}.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}

// Internal logs the underlying error and answers with an opaque 500.
func Internal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, "internal", "something went wrong")
}

func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, "invalid_request", message)
}

func UserID(c *gin.Context) string { return c.GetString("userID") }

// ParamUUID reads a path parameter and rejects anything that is not a UUID.
func ParamUUID(c *gin.Context, name string) (string, bool) {
	id := strings.TrimSpace(c.Param(name))
	if _, err := uuid.Parse(id); err != nil {
		Fail(c, http.StatusBadRequest, "invalid_id", "invalid identifier")
		return "", false
	}
	return strings.ToLower(id), true
}

// QueryInt parses an optional integer query parameter and clamps it to [min, max].
func QueryInt(c *gin.Context, name string, def, min, max int) (int, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return def, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min || v > max {
		Fail(c, http.StatusBadRequest, "invalid_query", "invalid query parameter: "+name)
		return 0, false
	}
	return v, true
}
