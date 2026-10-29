// Package httpx holds small helpers shared by Gin handlers.
package httpx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

const maxJSONBody = 64 << 10

// Fail writes a JSON error response translated from err. Unknown errors are logged
// and reported as a generic 500 so internals never leak to clients.
func Fail(c *gin.Context, operation string, err error) {
	status := http.StatusInternalServerError
	message := "internal error"

	var appErr *platformerrors.AppError
	if errors.As(err, &appErr) {
		message = appErr.Message
		switch {
		case errors.Is(appErr.Kind, platformerrors.ErrInvalid):
			status = http.StatusBadRequest
		case errors.Is(appErr.Kind, platformerrors.ErrUnauthorized):
			status = http.StatusUnauthorized
		case errors.Is(appErr.Kind, platformerrors.ErrForbidden):
			status = http.StatusForbidden
		case errors.Is(appErr.Kind, platformerrors.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(appErr.Kind, platformerrors.ErrConflict):
			status = http.StatusConflict
		case errors.Is(appErr.Kind, platformerrors.ErrRateLimited):
			status = http.StatusTooManyRequests
		}
	}
	if status >= 500 {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// BindJSON decodes a size-limited JSON body into dst.
func BindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBody)
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, "bind", platformerrors.Invalid("invalid request"))
		return false
	}
	return true
}

// Limit parses the ?limit= query parameter, applying a default and a hard maximum.
func Limit(c *gin.Context, def, max int) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		Fail(c, "limit", platformerrors.Invalid("limit must be between 1 and %d", max))
		return 0, false
	}
	return n, true
}

// NoStore marks a response as non-cacheable (private data).
func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}
