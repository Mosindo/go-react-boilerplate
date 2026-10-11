// Package httpx holds small Gin helpers shared by every feature handler.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: message, Code: code})
}

func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, "invalid_request", message)
}

// Respond writes err as an AppError response, or a generic 500 (logged) for anything else.
func Respond(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		Fail(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, "internal_error", "something went wrong")
}

func UserID(c *gin.Context) string { return c.GetString("userID") }

// Limit parses the `limit` query parameter, clamped to [1, max].
func Limit(c *gin.Context, def, max int) int {
	raw := strings.TrimSpace(c.Query("limit"))
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

// UUIDParam returns the named path parameter if it is a valid UUID, otherwise writes a 404.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := c.Param(name)
	if _, err := uuid.Parse(value); err != nil {
		Fail(c, http.StatusNotFound, "not_found", "not found")
		return "", false
	}
	return value, true
}
