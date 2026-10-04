// Package httpx holds the small helpers shared by every Gin handler.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

const maxJSONBody = 64 << 10

type errorBody struct {
	Error  string            `json:"error"`
	Code   string            `json:"code"`
	Fields map[string]string `json:"fields,omitempty"`
}

var statusByKind = map[platformerrors.Kind]int{
	platformerrors.KindValidation:   http.StatusBadRequest,
	platformerrors.KindUnauthorized: http.StatusUnauthorized,
	platformerrors.KindForbidden:    http.StatusForbidden,
	platformerrors.KindNotFound:     http.StatusNotFound,
	platformerrors.KindConflict:     http.StatusConflict,
	platformerrors.KindTooLarge:     http.StatusRequestEntityTooLarge,
	platformerrors.KindRateLimited:  http.StatusTooManyRequests,
}

// Fail writes a consistent error response. Unexpected errors are logged and hidden behind a 500.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		status := statusByKind[appErr.Kind]
		if status == 0 {
			status = http.StatusInternalServerError
		}
		c.AbortWithStatusJSON(status, errorBody{Error: appErr.Message, Code: appErr.Code, Fields: appErr.Fields})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody{Error: "Something went wrong", Code: "internal"})
}

// BindJSON decodes a size-limited JSON body, rejecting unknown trailing data.
func BindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBody)
	dec := json.NewDecoder(c.Request.Body)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			Fail(c, "bind", platformerrors.TooLarge("Request body too large"))
			return false
		}
		Fail(c, "bind", platformerrors.Validation("Invalid JSON body", nil))
		return false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		Fail(c, "bind", platformerrors.Validation("Invalid JSON body", nil))
		return false
	}
	return true
}

func UserID(c *gin.Context) string { return c.GetString("userID") }

// Limit parses ?limit= with bounds (default 20, max 50).
func Limit(c *gin.Context) (int, error) {
	return LimitWith(c, 20, 50)
}

func LimitWith(c *gin.Context, def, max int) (int, error) {
	raw := c.Query("limit")
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		return 0, platformerrors.FieldInvalid("limit", "must be between 1 and "+strconv.Itoa(max))
	}
	return n, nil
}

func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }
