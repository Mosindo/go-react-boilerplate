// Package httpx holds the small HTTP helpers shared by every feature handler:
// the uniform error body and strict, size-limited JSON binding.
package httpx

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Error is the body of every non-2xx response.
type Error struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

const (
	CodeInvalidRequest     = "invalid_request"
	CodeUnauthorized       = "unauthorized"
	CodeForbidden          = "forbidden"
	CodeNotFound           = "not_found"
	CodeConflict           = "conflict"
	CodeRateLimited        = "rate_limited"
	CodeProfileIncomplete  = "profile_incomplete"
	CodeInternal           = "internal"
	maxJSONBodyBytes       = 64 << 10
	DefaultLimit           = 20
	MaxLimit               = 50
	defaultPaginationStart = 0
)

// Fail writes the uniform error body and aborts the request.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, Error{Error: message, Code: code})
}

func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, CodeInvalidRequest, message)
}

func NotFound(c *gin.Context) {
	Fail(c, http.StatusNotFound, CodeNotFound, "not found")
}

func Forbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, CodeForbidden, message)
}

func Internal(c *gin.Context) {
	Fail(c, http.StatusInternalServerError, CodeInternal, "internal error")
}

// BindJSON decodes a size-limited JSON body into dst and replies 400 on failure.
func BindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)
	if err := c.ShouldBindJSON(dst); err != nil {
		BadRequest(c, "invalid request body")
		return false
	}
	return true
}

// Pagination parses limit/offset query parameters with strict bounds.
func Pagination(c *gin.Context, defaultLimit int) (limit, offset int, ok bool) {
	limit = defaultLimit
	offset = defaultPaginationStart
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > MaxLimit {
			BadRequest(c, "limit must be between 1 and 50")
			return 0, 0, false
		}
		limit = v
	}
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 || v > 100000 {
			BadRequest(c, "offset must be a non-negative integer")
			return 0, 0, false
		}
		offset = v
	}
	return limit, offset, true
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical UUID. Handlers use it to turn a
// malformed identifier into a clean 404 before it reaches the database.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }
