// Package httpx holds the shared JSON error contract: {"error": "<message>", "code": "<machine_code>"}.
package httpx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}

func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, "invalid_request", message)
}
func NotFound(c *gin.Context) { Fail(c, http.StatusNotFound, "not_found", "not found") }
func Internal(c *gin.Context) { Fail(c, http.StatusInternalServerError, "internal", "internal error") }

// Pagination parses limit/offset query params with strict bounds.
func Pagination(c *gin.Context, defLimit, maxLimit int) (limit, offset int, ok bool) {
	limit, offset = defLimit, 0
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > maxLimit {
			BadRequest(c, "invalid limit")
			return 0, 0, false
		}
		limit = v
	}
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 || v > 100000 {
			BadRequest(c, "invalid offset")
			return 0, 0, false
		}
		offset = v
	}
	return limit, offset, true
}
