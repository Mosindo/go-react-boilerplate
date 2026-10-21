// Package httpx holds Gin helpers shared by every feature handler.
package httpx

import (
	"net/http"
	"strconv"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes a consistent JSON error. Unknown errors are logged and masked.
func Fail(c *gin.Context, operation string, err error) {
	if be, ok := perr.As(err); ok {
		c.AbortWithStatusJSON(statusFor(be.Kind), errorBody{Error: be.Message, Code: be.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody{Error: "internal error", Code: "internal"})
}

// BadRequest answers malformed payloads (JSON syntax, wrong types).
func BadRequest(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, errorBody{Error: "invalid request", Code: "invalid_request"})
}

func statusFor(k perr.Kind) int {
	switch k {
	case perr.KindInvalid:
		return http.StatusBadRequest
	case perr.KindUnauthorized:
		return http.StatusUnauthorized
	case perr.KindForbidden:
		return http.StatusForbidden
	case perr.KindNotFound:
		return http.StatusNotFound
	case perr.KindConflict:
		return http.StatusConflict
	case perr.KindTooManyRequests:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string { return c.GetString("userID") }

// Bind decodes a JSON body, answering 400 itself on failure.
func Bind(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	if err := c.ShouldBindJSON(dst); err != nil {
		BadRequest(c)
		return false
	}
	return true
}

// Limit parses ?limit= with bounds.
func Limit(c *gin.Context, def, max int) int {
	n, err := strconv.Atoi(c.Query("limit"))
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// NoStore marks a response as non cacheable (tokens, private data).
func NoStore(c *gin.Context) { c.Header("Cache-Control", "no-store") }
