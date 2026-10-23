// Package httpx holds the small helpers shared by every Gin handler.
package httpx

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

const maxJSONBody = 1 << 20

// BindJSON decodes a size-limited JSON body. It writes the error response itself.
func BindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBody)
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, platformerrors.BadRequest("invalid_request", "invalid request body"))
		return false
	}
	return true
}

// Fail writes an error response. Unknown errors are logged and masked as 500.
func Fail(c *gin.Context, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		body := gin.H{"error": appErr.Message, "code": appErr.Code}
		if len(appErr.Fields) > 0 {
			body["fields"] = appErr.Fields
		}
		c.AbortWithStatusJSON(appErr.Status, body)
		return
	}
	log.Printf(`{"event":"internal_error","request_id":%q,"path":%q,"error":%q}`,
		c.GetString("request_id"), c.FullPath(), err.Error())
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal_error"})
}

func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }

func UserID(c *gin.Context) string { return c.GetString("userID") }

// IsUUID reports whether s is a canonical textual UUID.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if s[i] != '-' {
				return false
			}
		case (s[i] >= '0' && s[i] <= '9') || (s[i] >= 'a' && s[i] <= 'f') || (s[i] >= 'A' && s[i] <= 'F'):
		default:
			return false
		}
	}
	return true
}

// ParamUUID reads a UUID path parameter or answers 404.
func ParamUUID(c *gin.Context, name string) (string, bool) {
	v := strings.ToLower(c.Param(name))
	if !IsUUID(v) {
		Fail(c, platformerrors.NotFound("not_found", "resource not found"))
		return "", false
	}
	return v, true
}

// QueryInt reads a bounded integer query parameter, falling back to def when absent.
func QueryInt(c *gin.Context, name string, def, min, max int) (int, bool) {
	raw := c.Query(name)
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min || n > max {
		Fail(c, platformerrors.BadRequest("invalid_request", name+" is out of range"))
		return 0, false
	}
	return n, true
}
