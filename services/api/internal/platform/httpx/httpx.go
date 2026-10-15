// Package httpx holds the small set of helpers shared by every feature handler.
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

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes the JSON error response for err. Business errors keep their
// status and code; anything else is logged and reported as an opaque 500.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		if appErr.Status >= http.StatusInternalServerError {
			logger.LogHandlerError(c, operation, appErr.Status, err)
		}
		c.AbortWithStatusJSON(appErr.Status, errorBody{Error: appErr.Message, Code: appErr.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody{Error: "internal error", Code: "internal"})
}

// Bind decodes the JSON body into dst and reports malformed input as a 400.
func Bind(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, "bind", platformerrors.BadRequest("invalid_request", "invalid request body"))
		return false
	}
	return true
}

// Page parses limit/offset query parameters. Invalid values are rejected and
// limits above maxLimit are clamped.
func Page(c *gin.Context, defaultLimit, maxLimit int) (limit, offset int, ok bool) {
	limit = defaultLimit
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			Fail(c, "page", platformerrors.BadRequest("invalid_limit", "limit must be a positive integer"))
			return 0, 0, false
		}
		limit = v
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			Fail(c, "page", platformerrors.BadRequest("invalid_offset", "offset must be a non-negative integer"))
			return 0, 0, false
		}
		offset = v
	}
	return limit, offset, true
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// NoStore marks a response as non-cacheable (tokens, private data).
func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical UUID, so path parameters can be
// rejected before they reach a uuid-typed SQL parameter.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }
